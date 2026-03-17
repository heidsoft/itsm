package service

import (
	"fmt"
	"sync"
	"time"
)

// NumberGenerator 号码生成器
// 用于生成工单、事件、问题、变更的唯一编号
type NumberGenerator struct {
	mu           sync.Mutex
	ticketSeq    int
	incidentSeq  int
	problemSeq   int
	changeSeq    int
	sequenceDate string
}

// NewNumberGenerator 创建号码生成器
func NewNumberGenerator() *NumberGenerator {
	return &NumberGenerator{
		ticketSeq:    0,
		incidentSeq:  0,
		problemSeq:   0,
		changeSeq:    0,
		sequenceDate: time.Now().Format("20060102"),
	}
}

// generateSequence 生成序列号
// 每天从0001开始，重置日期时序列也会重置
func (ng *NumberGenerator) generateSequence(seqType string) int {
	ng.mu.Lock()
	defer ng.mu.Unlock()

	today := time.Now().Format("20060102")

	// 检查是否需要重置序列
	if today != ng.sequenceDate {
		ng.sequenceDate = today
		ng.ticketSeq = 0
		ng.incidentSeq = 0
		ng.problemSeq = 0
		ng.changeSeq = 0
	}

	// 根据类型递增对应序列
	switch seqType {
	case "ticket":
		ng.ticketSeq++
		return ng.ticketSeq
	case "incident":
		ng.incidentSeq++
		return ng.incidentSeq
	case "problem":
		ng.problemSeq++
		return ng.problemSeq
	case "change":
		ng.changeSeq++
		return ng.changeSeq
	default:
		return 0
	}
}

// GenerateTicketNumber 生成工单编号
// 格式: TK-YYYYMMDD-XXXX
func (ng *NumberGenerator) GenerateTicketNumber() string {
	seq := ng.generateSequence("ticket")
	return fmt.Sprintf("TK-%s-%04d", ng.sequenceDate, seq)
}

// GenerateIncidentNumber 生成事件编号
// 格式: INC-YYYYMMDD-XXXX
func (ng *NumberGenerator) GenerateIncidentNumber() string {
	seq := ng.generateSequence("incident")
	return fmt.Sprintf("INC-%s-%04d", ng.sequenceDate, seq)
}

// GenerateProblemNumber 生成问题编号
// 格式: PRB-YYYYMMDD-XXXX
func (ng *NumberGenerator) GenerateProblemNumber() string {
	seq := ng.generateSequence("problem")
	return fmt.Sprintf("PRB-%s-%04d", ng.sequenceDate, seq)
}

// GenerateChangeNumber 生成变更编号
// 格式: CHG-YYYYMMDD-XXXX
func (ng *NumberGenerator) GenerateChangeNumber() string {
	seq := ng.generateSequence("change")
	return fmt.Sprintf("CHG-%s-%04d", ng.sequenceDate, seq)
}

// 全局号码生成器实例
var globalGenerator = NewNumberGenerator()

// GenerateTicketNumberGlobal 全局工单编号生成
func GenerateTicketNumberGlobal() string {
	return globalGenerator.GenerateTicketNumber()
}

// GenerateIncidentNumberGlobal 全局事件编号生成
func GenerateIncidentNumberGlobal() string {
	return globalGenerator.GenerateIncidentNumber()
}

// GenerateProblemNumberGlobal 全局问题编号生成
func GenerateProblemNumberGlobal() string {
	return globalGenerator.GenerateProblemNumber()
}

// GenerateChangeNumberGlobal 全局变更编号生成
func GenerateChangeNumberGlobal() string {
	return globalGenerator.GenerateChangeNumber()
}
