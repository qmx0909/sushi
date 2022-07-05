package main

import (
	"container/list"
	"fmt"
	"gyrus_sushi/pojo"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

//m 寿司店师傅总数
//n 寿司店可容纳顾客总数
//N 寿司店传输带上最多可放置的寿司盘数
//SushiMaterialCount 寿司原材料数量 可做多少盘寿司
//NewCustomerInterval 新顾客到店间隔(秒)
//ProduceInterval 寿司师傅制作寿司间隔(秒)
//CustomInterval 顾客吃寿司间隔(秒)
//CheckInterval 检查间隔(秒)
//SushiChefProduceBase 寿司师傅最低生产寿司数
//SushiChefProduceIncreaseArea 寿司师傅生产寿司差异区间
//SushiChefProduceSpeedArea 寿司师傅生产寿司速度区间
//CustomerCustomBase 顾客最小食量
//CustomerCustomIncreaseArea 顾客食量差异区间
//CustomerCustomSpeedArea 顾客消费寿司速度区间
const (
	m, n                         = 5, 10
	N                            = 20
	SushiMaterialCount           = 100
	NewCustomerInterval          = 10 * time.Second
	ProduceInterval              = 10 * time.Second
	CustomInterval               = 10 * time.Second
	CheckInterval                = 1 * time.Second
	SushiChefProduceBase         = 10
	SushiChefProduceIncreaseArea = 10
	SushiChefProduceSpeedArea    = 3
	CustomerCustomBase           = 3
	CustomerCustomIncreaseArea   = 3
	CustomerCustomSpeedArea      = 3
)

func main() {
	//随机数生成种子
	rand.Seed(time.Now().UnixNano())

	//服务总顾客数
	var customerSum int32
	//寿司原材料锁
	var sushiMaterialMutex sync.Mutex
	//传输带寿司锁
	var sushiOnBeltMutex sync.Mutex
	//寿司师傅waitGroup
	var sushiChefWaitGroup sync.WaitGroup
	//寿司店waitGroup
	var wg sync.WaitGroup

	//寿司师傅数组初始化
	scs := list.New()
	//顾客数组初始化
	cs := list.New()
	//寿司师傅生产寿司通道
	sushiCountChannel := make(chan int32)
	//放置传送带寿司通道
	sushiOnBeltCountChannel := make(chan int32)
	//寿司师傅下班通道
	sushiChefLeaveChannel := make(chan *pojo.SushiChef)
	//顾客离店通道
	customerLeaveChannel := make(chan *pojo.Customer)

	//初始化回转寿司店
	sb := pojo.SushiBar{
		SushiChefs:                 scs,
		Customers:                  cs,
		CurrentSushiMaterialCount:  SushiMaterialCount,
		SushiOnBeltCountUpperLimit: N,
		CurrentSushiOnBeltCount:    0,
		IsOpen:                     true,
		SushiBox:                   0,
		IsAllSushiChefLeaveWork:    false,
	}
	fmt.Printf("寿司店开始营业，当前寿司原材料可制作寿司%d盘\n", SushiMaterialCount)

	//初始化寿司师傅
	var i int32
	for i = 0; i < m; i++ {
		sushiChef := pojo.SushiChef{
			Sid:                    i,
			Name:                   "寿司师傅" + strconv.Itoa(int(i)),
			ProductionSpeed:        rand.Int31n(SushiChefProduceSpeedArea) + 1,
			ProductionUpperLimit:   SushiChefProduceBase + rand.Int31n(SushiChefProduceIncreaseArea) + 1,
			CurrentProductionCount: 0,
		}
		scs.PushBack(sushiChef)
		fmt.Printf("寿司师傅%d已到店,当前师傅可生产寿司%d盘/分钟,最多可生产%d盘\n", sushiChef.Sid, sushiChef.ProductionSpeed, sushiChef.ProductionUpperLimit)
	}

	//寿司师傅工作goroutine
	for e := scs.Front(); e != nil; e = e.Next() {
		sushiChef := e.Value.(pojo.SushiChef)
		sushiChefWaitGroup.Add(1)
		go sushiChef.Produce(sushiCountChannel, sushiMaterialMutex, &sb, sushiChefWaitGroup, sushiChefLeaveChannel, ProduceInterval)
	}

	//寿司师傅下班goroutine
	wg.Add(1)
	go func() {
		for {
			if sb.SushiChefs.Len() == 0 {
				sb.IsAllSushiChefLeaveWork = true
				fmt.Println("所有寿司师傅均已下班，不再生产寿司")
				break
			}
			select {
			case sushiChef := <-sushiChefLeaveChannel:
				var next *list.Element
				for e := sb.SushiChefs.Front(); e != nil; e = next {
					next = e.Next()
					if e.Value.(pojo.SushiChef).Sid == sushiChef.Sid {
						sb.SushiChefs.Remove(e)
						break
					}
				}
				fmt.Printf("%s已下班\n", sushiChef.Name)
			default:
			}
			time.Sleep(CheckInterval)
		}
		wg.Done()
	}()

	//接收寿司师傅生产的寿司，并向传输带上放置寿司goroutine
	wg.Add(1)
	go func() {
		for {
			//退出条件，所有寿司师傅都下班
			if sb.IsAllSushiChefLeaveWork {
				break
			}
			//从channel中获取生产的寿司数量
			select {
			case produceSushiCount := <-sushiCountChannel:
				sushiOnBeltCountChannel <- produceSushiCount
			default:
			}
			time.Sleep(CheckInterval)
		}
		wg.Done()
	}()

	//处理传输带上的寿司goroutine
	wg.Add(1)
	go func() {
		//寿司箱子，用于缓存多余的寿司
		sb.SushiBox = 0
		for {
			//退出条件，所有寿司师傅都下班，且寿司箱子为空
			if sb.IsAllSushiChefLeaveWork && sb.SushiBox == 0 {
				fmt.Println("所有寿司师傅均已下班，寿司盒子仓库为空，不再向传输带上放置寿司")
				break
			}

			//从channel中获取送来的寿司，并缓存到寿司盒子中
			select {
			case sendSushiCount := <-sushiOnBeltCountChannel:
				sb.SushiBox += sendSushiCount
			default:
			}

			//计算并Add待补充到传输带上的寿司数量
			sushiOnBeltMutex.Lock()
			if sb.CurrentSushiOnBeltCount < N {
				toAddSushiCount := N - sb.CurrentSushiOnBeltCount
				if toAddSushiCount > sb.SushiBox {
					toAddSushiCount = sb.SushiBox
					sb.SushiBox = 0
				} else {
					sb.SushiBox -= toAddSushiCount
				}
				if toAddSushiCount > 0 {
					sb.CurrentSushiOnBeltCount += toAddSushiCount
					fmt.Printf("向传输带补充%d盘寿司，当前传送带上共%d盘寿司，传送带最大可放置%d盘寿司，当前盒子剩余%d盘寿司\n", toAddSushiCount, sb.CurrentSushiOnBeltCount, N, sb.SushiBox)
				}
			}
			sushiOnBeltMutex.Unlock()
			time.Sleep(CheckInterval)
		}
		wg.Done()
	}()

	//模拟顾客到店就餐goroutine
	var hasNewCustomer bool = true
	wg.Add(1)
	go func() {
		//顾客id
		var cid int32 = 0
		for {
			//寿司店营业中，来顾客
			if !sb.IsClose() {
				//顾客未满
				if sb.Customers.Len() < int(n) {
					c := pojo.Customer{
						Cid:             cid,
						Name:            "顾客" + strconv.Itoa(int(cid)),
						EatSpeed:        rand.Int31n(CustomerCustomSpeedArea) + 1,
						EatUpperLimit:   CustomerCustomBase + rand.Int31n(CustomerCustomIncreaseArea) + 1,
						CurrentEatCount: 0,
					}
					cs.PushBack(c)
					customerSum++
					fmt.Printf("%s已到店开始消费\n", c.Name)
					//顾客开始消费
					c.Custom(customerLeaveChannel, sushiOnBeltMutex, &sb, CustomInterval)
					cid++
				}
			} else {
				fmt.Println("回转寿司店已打烊")
				hasNewCustomer = false
				break
			}
			time.Sleep(NewCustomerInterval)
		}
		wg.Done()
	}()

	//处理顾客消费goroutine
	wg.Add(1)
	go func() {
		for {
			//当寿司店打烊且没有新顾客来且已有顾客全部离店，结束处理
			if sb.IsClose() && !hasNewCustomer && sb.Customers.Len() == 0 {
				fmt.Println("所有顾客均已离店")
				break
			}

			select {
			case c := <-customerLeaveChannel:
				var next *list.Element
				for e := sb.Customers.Front(); e != nil; e = next {
					next = e.Next()
					if e.Value.(pojo.Customer).Cid == c.Cid {
						sb.Customers.Remove(e)
						break
					}
				}
				fmt.Printf("%s已离店，共消费%d盘寿司，食量%d盘寿司\n", c.Name, c.CurrentEatCount, c.EatUpperLimit)
			default:
			}
			time.Sleep(CheckInterval)
		}
		wg.Done()
	}()

	//等待所有goroutine结束
	wg.Wait()
	// 关门后统计剩余寿司数量为寿司盒子中剩余数量与传输带上数量之和
	fmt.Printf("回转寿司店当天营业结束，共服务%d位顾客，剩余%d盘寿司\n", customerSum, sb.SushiBox+sb.CurrentSushiOnBeltCount)
}
