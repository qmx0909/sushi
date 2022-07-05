package pojo

import (
	"fmt"
	"sync"
	"time"
)

// Customer 顾客
type Customer struct {
	Cid             int32  //顾客id
	Name            string //顾客姓名
	EatSpeed        int32  //进食速度 盘/分钟
	EatUpperLimit   int32  //食量上限 盘
	CurrentEatCount int32  //当前已经吃过的数量 盘
}

// Custom 消费
func (c *Customer) Custom(customerLeaveChannel chan *Customer, sushiOnBeltMutex sync.Mutex, sushiBar *SushiBar, customInterval time.Duration) {
	go func() {
		for {
			sushiOnBeltMutex.Lock()
			if c.isStuffed() {
				//吃饱离店
				customerLeaveChannel <- c
				sushiOnBeltMutex.Unlock()
				break
			} else if sushiBar.CurrentSushiOnBeltCount == 0 && sushiBar.IsAllSushiChefLeaveWork {
				//未吃饱离店
				customerLeaveChannel <- c
				sushiOnBeltMutex.Unlock()
				break
			}
			c.Eat(sushiBar)
			sushiOnBeltMutex.Unlock()

			time.Sleep(customInterval)
		}
	}()
}

//是否吃饱
func (c *Customer) isStuffed() bool {
	if c.CurrentEatCount < c.EatUpperLimit {
		return false
	}
	return true
}

// Eat 吃寿司
func (c *Customer) Eat(sushiBar *SushiBar) {
	var customCount int32
	if c.EatSpeed <= sushiBar.CurrentSushiOnBeltCount {
		// 本次想吃的寿司不超过当前传输带上总寿司数
		// 判断本地吃的寿司数是否超过食量
		if c.EatSpeed <= (c.EatUpperLimit - c.CurrentEatCount) {
			customCount = c.EatSpeed
		} else {
			customCount = c.EatUpperLimit - c.CurrentEatCount
		}
	} else {
		// 本次想吃的寿司数超过当前传送带上总寿司数
		customCount = sushiBar.CurrentSushiOnBeltCount
		if customCount > (c.EatUpperLimit - c.CurrentEatCount) {
			customCount = c.EatUpperLimit - c.CurrentEatCount
		}
	}
	sushiBar.CurrentSushiOnBeltCount -= customCount
	c.CurrentEatCount += customCount
	fmt.Printf("%s本轮消费%d盘寿司，累计消费%d盘寿司，食量%d盘寿司\n", c.Name, customCount, c.CurrentEatCount, c.EatUpperLimit)
}
