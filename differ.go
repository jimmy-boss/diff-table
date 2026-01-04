// Package difftable
//
// ----------------develop info----------------
//
//	@Author Jimmy
//	@DateTime 2026-05-22 15:10
//
// --------------------------------------------
package difftable

import (
	"errors"

	"github.com/jimmy-boss/go-log/glog"
	"gorm.io/gorm"
)

// Option 配置函数类型
type Option func(*Differ)

// Differ 核心对比器
type Differ struct {
	srcDb     *gorm.DB
	dstDb     *gorm.DB
	tbPrefix  []string
	tbNames   []string
	diffMode  DiffMode
	outputFmt OutputFormat
	srcDbName string
	dstDbName string
	logger    glog.HLoggerBase
}

// WithSrcDb 注入源库 GORM 连接
func WithSrcDb(db *gorm.DB) Option {
	return func(d *Differ) {
		d.srcDb = db
	}
}

// WithDstDb 注入目标库 GORM 连接
func WithDstDb(db *gorm.DB) Option {
	return func(d *Differ) {
		d.dstDb = db
	}
}

// WithTbPrefix 设置表前缀白名单（可多个，OR 关系）
func WithTbPrefix(prefix ...string) Option {
	return func(d *Differ) {
		d.tbPrefix = append(d.tbPrefix, prefix...)
	}
}

// WithTbName 精确指定对比的表名（可多个，OR 关系）
func WithTbName(names ...string) Option {
	return func(d *Differ) {
		d.tbNames = append(d.tbNames, names...)
	}
}

// WithDiffMode 设置差异模式
func WithDiffMode(mode DiffMode) Option {
	return func(d *Differ) {
		d.diffMode = mode
	}
}

// WithOutputFormat 设置输出格式
func WithOutputFormat(fmt OutputFormat) Option {
	return func(d *Differ) {
		d.outputFmt = fmt
	}
}

// WithSrcDbName 覆盖源库默认库名（db_name.table_name 场景）
func WithSrcDbName(dbName string) Option {
	return func(d *Differ) {
		d.srcDbName = dbName
	}
}

// WithDstDbName 覆盖目标库默认库名
func WithDstDbName(dbName string) Option {
	return func(d *Differ) {
		d.dstDbName = dbName
	}
}

// WithLogger 注入日志实例
func WithLogger(logger glog.HLoggerBase) Option {
	return func(d *Differ) {
		d.logger = logger
	}
}

// NewDiffer 构造函数
func NewDiffer(opts ...Option) *Differ {
	d := &Differ{
		diffMode:  DiffOnly,
		outputFmt: OutputJSON,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

// validate 校验必要参数
func (d *Differ) validate() error {
	if d.srcDb == nil {
		return errors.New("srcDb is required, use WithSrcDb to set")
	}
	if d.dstDb == nil {
		return errors.New("dstDb is required, use WithDstDb to set")
	}
	return nil
}
