package model

/*
DailyTrend 是看板趋势和分布统计的通用返回结构。
Date 字段通常表示日期，在资产类型分布场景中也会复用为类型名称。
*/
type DailyTrend struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}
