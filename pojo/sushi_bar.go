package pojo

import "container/list"

// SushiBar 回转寿司店
type SushiBar struct {
	SushiChefs                 *list.List // 寿司师傅数组
	Customers                  *list.List // 顾客数组
	CurrentSushiMaterialCount  int32      //当前寿司原材量数量
	SushiOnBeltCountUpperLimit int32      //传输带可放置的最大寿司数量
	CurrentSushiOnBeltCount    int32      //当前传输带上的寿司数量
	IsOpen                     bool       //是否在营业中
	SushiBox                   int32      //寿司盒子中寿司数量，用于储存多余寿司
	IsAllSushiChefLeaveWork    bool       //是否所有寿司师傅均下班
}

// IsClose 是否打烊，打烊条件：所有寿司师傅均下班，且寿司盒子库存为0（忽略传输带上还有寿司的情况）
func (sb SushiBar) IsClose() bool {
	if sb.IsAllSushiChefLeaveWork && sb.SushiBox == 0 {
		return true
	}
	return false
}
