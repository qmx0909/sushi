package pojo

import (
	"fmt"
	"sync"
	"time"
)

// SushiChef 寿司师傅
type SushiChef struct {
	Sid                    int32  //寿司师傅id
	Name                   string //名字
	ProductionSpeed        int32  //生产速度 盘/分钟
	ProductionUpperLimit   int32  //生产寿司上限 盘
	CurrentProductionCount int32  //当前已生产寿司盘数
}

// IsFinished 是否做完定量寿司
func (s *SushiChef) IsFinished() bool {
	if s.CurrentProductionCount < s.ProductionUpperLimit {
		return false
	}
	return true
}

// Produce 生产寿司
func (s *SushiChef) Produce(sushiCountChannel chan int32, mutex sync.Mutex, sushiBar *SushiBar, wg sync.WaitGroup, sushiChefLeaveChannel chan *SushiChef, produceInterval time.Duration) {
	go func() {
		for {
			if s.IsFinished() {
				sushiChefLeaveChannel <- s
				fmt.Printf("%s已做完定量寿司，下班\n", s.Name)
				break
			}

			//按照寿司制作速度，判断本次的寿司制作数量是否会导致当前师傅的寿司总制作量超过该师傅的上限
			//当该情况发生时，本次寿司的制作数量为当前总制作量与上限的差值
			var produceCount int32
			if (s.CurrentProductionCount + s.ProductionSpeed) > s.ProductionUpperLimit {
				produceCount = s.ProductionUpperLimit - s.CurrentProductionCount
			} else {
				produceCount = s.ProductionSpeed
			}

			mutex.Lock()
			//判断原材料是否耗尽
			if sushiBar.CurrentSushiMaterialCount == 0 {
				sushiChefLeaveChannel <- s
				fmt.Printf("原材料已耗尽，%s下班\n", s.Name)
				mutex.Unlock()
				break
			}
			//判断寿司店剩余原材料数量是否够本次师傅制作
			if sushiBar.CurrentSushiMaterialCount < produceCount {
				produceCount = sushiBar.CurrentSushiMaterialCount
			}
			sushiBar.CurrentSushiMaterialCount -= produceCount
			mutex.Unlock()

			sushiCountChannel <- produceCount
			s.CurrentProductionCount = +produceCount
			fmt.Printf("%s制作了%d盘寿司\n", s.Name, produceCount)

			time.Sleep(produceInterval)
		}
		wg.Done()
	}()
}
