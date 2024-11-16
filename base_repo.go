package gormx

import (
	"context"
	"go/ast"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils"
)

type BaseRepo[T any] struct {
	GormDB *gorm.DB
	// 泛型参数代表的struct名称，例如：BaseRepo[Pop]
	StructName string
	PrimaryKey string
}

// NewBaseRepo 这个函数的意义在于不暴露db进行初始化，外部只能通过函数DB()获取
func NewBaseRepo[T any](db *gorm.DB) BaseRepo[T] {
	b := BaseRepo[T]{
		GormDB: db,
	}
	var m T
	b.StructName = reflect.ValueOf(m).Type().Name()
	b.PrimaryKey = b.parsePrimaryKey()
	return b
}

// WithTransactionCtx 和事务相关的db操作，在取db连接时均采用此方法
func (b *BaseRepo[T]) withTransactionCtx(ctx context.Context) *gorm.DB {
	tx, ok := ctx.Value(contextTxKey{}).(*gorm.DB)
	if ok {
		return tx
	}
	return b.GormDB.WithContext(ctx)
}

// InTx fn是包含了事务操作的方法，只要fn里面有异常，里面的db操作都会回滚
func (b *BaseRepo[T]) InTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return b.GormDB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, contextTxKey{}, tx)
		return fn(ctx)
	})
}

func (b *BaseRepo[T]) parsePrimaryKey() string {
	var m T
	return recursiveParsePrimaryKey(reflect.ValueOf(m))
}

func recursiveParsePrimaryKey(reflectValue reflect.Value) string {
	reflectType := IndirectType(reflectValue.Type())
	var hasId bool
	for i := 0; i < reflectType.NumField(); i++ {
		if fieldStruct := reflectType.Field(i); ast.IsExported(fieldStruct.Name) {
			if fieldStruct.Anonymous {
				res := recursiveParsePrimaryKey(reflectValue.Field(i))
				if res != "" {
					return res
				}
			} else {
				tagSetting := schema.ParseTagSetting(fieldStruct.Tag.Get("gorm"), ";")

				// 数据库字段名
				columnName := tagSetting["COLUMN"]
				if columnName == "" {
					columnName = schema.NamingStrategy{}.ColumnName("", fieldStruct.Name)
				}

				if utils.CheckTruth(tagSetting["PRIMARYKEY"], tagSetting["PRIMARY_KEY"]) {
					return columnName
				}

				if columnName == "id" {
					hasId = true
				}
			}
		}
	}
	if hasId {
		return "id"
	}
	return ""
}

// Insert 插入单条记录
func (b *BaseRepo[T]) Insert(ctx context.Context, m *T) (err error) {
	if err = b.withTransactionCtx(ctx).Create(m).Error; err != nil {
		err = errors.Wrapf(err, "db: insert %s error, param: %+v", b.StructName, m)
	}
	return
}

// BatchInsert 批量插入
// 注：需要根据插入数据的大小来设置batchSize
func (b *BaseRepo[T]) BatchInsert(ctx context.Context, m []*T, batchSize int) (int64, error) {
	tx := b.withTransactionCtx(ctx).CreateInBatches(m, batchSize)
	if tx.Error != nil {
		return 0, errors.Wrapf(tx.Error, "db: batch insert %s error, param: %+v", b.StructName, m)
	}
	return tx.RowsAffected, nil
}

// DeleteByPK 根据主键删除，支持单个主键或者一个主键数组
func (b *BaseRepo[T]) DeleteByPK(ctx context.Context, pks any) (int64, error) {
	var m T
	tx := b.withTransactionCtx(ctx).Where(map[string]any{
		b.PrimaryKey: pks,
	}).Delete(&m)
	if err := tx.Error; err != nil {
		return 0, errors.Wrapf(err, "db: delete %s by pks error, pks: %v", b.StructName, pks)
	}
	return tx.RowsAffected, nil
}

// DeleteByMap 根据条件删除，支持零值
// condition示例：{"name","张三"}
// condition里的key兼容驼峰和蛇形
func (b *BaseRepo[T]) DeleteByMap(ctx context.Context, condition map[string]any) (int64, error) {
	c := camel2SnakeForMapKey(condition)
	var m T
	tx := b.withTransactionCtx(ctx).Where(c).Delete(&m)
	if err := tx.Error; err != nil {
		return 0, errors.Wrapf(err, "db: delete %s by map error, condition: %v", b.StructName, condition)
	}
	return tx.RowsAffected, nil
}

// UpdateByPK 根据主键更新非空字段
func (b *BaseRepo[T]) UpdateByPK(ctx context.Context, t *T) (int64, error) {
	tx := b.withTransactionCtx(ctx).Model(t).Updates(t)
	if err := tx.Error; err != nil {
		return 0, errors.Wrapf(err, "db: update %s by pk error, param: %+v", b.StructName, t)
	}
	return tx.RowsAffected, nil
}

// UpdateByPKWithMap 根据id更新，支持零值
//
// updateData示例：{"age","18"}
//
// 注：这里会删除updateData里的以下字段
// 1、带有gorm标签：autoCreateTime、autoUpdateTime的字段
func (b *BaseRepo[T]) UpdateByPKWithMap(ctx context.Context, pk any, updateData map[string]any) (int64, error) {
	return b.UpdateByMap(ctx, map[string]any{b.PrimaryKey: pk}, updateData)
}

// UpdateByMap 根据条件更新，支持零值
//
// condition示例：{"name","张三"}
// condition里的key兼容驼峰和蛇形
//
// updateData示例：{"age","18"}
//
// 注：这里会删除updateData里的以下字段
// 1、带有gorm标签：autoCreateTime、autoUpdateTime的字段
func (b *BaseRepo[T]) UpdateByMap(ctx context.Context, condition map[string]any, updateData map[string]any) (int64, error) {
	c := camel2SnakeForMapKey(condition)
	b.deleteAutoTime(updateData)

	var m T
	tx := b.withTransactionCtx(ctx).Model(&m).Where(c).Updates(updateData)
	if err := tx.Error; err != nil {
		return 0, errors.Wrapf(err, "db: update %s by map error, condition: %v, updateData: %v", b.StructName, c, updateData)
	}
	return tx.RowsAffected, nil
}

func (b *BaseRepo[T]) deleteAutoTime(updateData map[string]any) {
	var m T
	b.recursiveDeleteAutoTime(reflect.ValueOf(m), updateData)
}

func (b *BaseRepo[T]) recursiveDeleteAutoTime(v reflect.Value, updateData map[string]any) {
	t := IndirectType(v.Type())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous {
			b.recursiveDeleteAutoTime(v.Field(i), updateData)
		} else {
			tag := field.Tag.Get("gorm")
			autoCreateTime := strings.Contains(tag, "autoCreateTime") &&
				!strings.Contains(tag, "autoCreateTime:false")

			autoUpdateTime := strings.Contains(tag, "autoUpdateTime") &&
				!strings.Contains(tag, "autoUpdateTime:false")
			if autoCreateTime || autoUpdateTime {
				delete(updateData, field.Name)
			}
		}
	}
}

// SelectOne 条件不能是零值，如果要查零值，请用 SelectOneByMap
func (b *BaseRepo[T]) SelectOne(ctx context.Context, condition *T) (*T, error) {
	return b.selectOne(ctx, condition)
}

// SelectOneByPK 根据主键查找
func (b *BaseRepo[T]) SelectOneByPK(ctx context.Context, pk any) (*T, error) {
	return b.SelectOneByMap(ctx, map[string]any{b.PrimaryKey: pk})
}

// SelectOneByMap 根据条件查找，支持零值
// condition示例：{"name","张三"}
// condition里的key兼容驼峰和蛇形
func (b *BaseRepo[T]) SelectOneByMap(ctx context.Context, condition map[string]any) (*T, error) {
	c := camel2SnakeForMapKey(condition)
	return b.selectOne(ctx, c)
}

func (b *BaseRepo[T]) selectOne(ctx context.Context, condition any) (*T, error) {
	res, err := b._select(ctx, condition)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}
	if len(res) > 1 {
		return nil, errors.Errorf("db: select one %s error, result must be one, now it is %d, condition %+v", b.StructName, len(res), condition)
	}
	return res[0], err
}

// Select 根据非空字段查询
func (b *BaseRepo[T]) Select(ctx context.Context, condition *T) ([]*T, error) {
	return b._select(ctx, condition)
}

// SelectAll 查询所有
func (b *BaseRepo[T]) SelectAll(ctx context.Context) ([]*T, error) {
	return b.SelectByMap(ctx, map[string]any{})
}

// SelectByPK 根据主键查找，支持单个主键或者一个主键数组
func (b *BaseRepo[T]) SelectByPK(ctx context.Context, pks any) ([]*T, error) {
	return b.SelectByMap(ctx, map[string]any{b.PrimaryKey: pks})
}

// SelectByMap 根据条件查找，支持零值
// condition示例：{"name","张三"}
// condition里的key兼容驼峰和蛇形
func (b *BaseRepo[T]) SelectByMap(ctx context.Context, condition map[string]any) ([]*T, error) {
	c := camel2SnakeForMapKey(condition)
	return b._select(ctx, c)
}

func camel2SnakeForMapKey(condition map[string]any) map[string]any {
	c := make(map[string]any)
	for k, v := range condition {
		c[Camel2Snake(k)] = v
	}
	return c
}

func (b *BaseRepo[T]) _select(ctx context.Context, condition any) ([]*T, error) {
	var (
		m   T
		res []*T
	)
	if err := b.withTransactionCtx(ctx).Model(&m).Where("deleted !=?", Deleted).Where(condition).Find(&res).Error; err != nil {
		return nil, errors.Wrapf(err, "db: select %s error, condition: %+v", b.StructName, condition)
	}
	return res, nil
}

type PageParam struct {
	PageNo   int32
	PageSize int32
	OrderBy  string
}

func (b *BaseRepo[T]) ListPage(_ context.Context, query *gorm.DB, page *PageParam) ([]*T, int32, error) {
	var (
		total int64
		res   []*T
	)
	query = query.Where("deleted !=?", Deleted)
	if page != nil {
		if err := query.Count(&total).Error; err != nil {
			return nil, 0, errors.Wrapf(err, "db: select count %s error", b.StructName)
		}
	}
	var orders []string
	if page != nil {
		query = query.Offset(int(page.PageNo-1) * int(page.PageSize)).Limit(int(page.PageSize))
		if page.OrderBy != "" {
			query = query.Order(page.OrderBy)
		}
	}
	if err := query.Find(&res).Error; err != nil {
		return nil, 0, errors.Wrapf(err, "db: select %s error, orders: %s", b.StructName, strings.Join(orders, ","))
	}
	if total == 0 {
		total = int64(len(res))
	}
	return res, int32(total), nil
}

func (b *BaseRepo[T]) PageSelect(ctx context.Context, page *PageParam, query any, args ...any) ([]*T, int32, error) {
	var (
		m     T
		total int64
		res   []*T
	)
	if page != nil {
		if err := b.withTransactionCtx(ctx).Model(&m).Where("deleted !=?", Deleted).Where(query, args...).Count(&total).Error; err != nil {
			return nil, 0, errors.Wrapf(err, "db: select count %s error, query: %+v, args: %+v", b.StructName, query, args)
		}
	}
	q := b.withTransactionCtx(ctx).Model(&m).Where("deleted !=?", Deleted).Where(query, args...)
	if page != nil {
		q = q.Offset(int(page.PageNo-1) * int(page.PageSize)).Limit(int(page.PageSize))
		if page.OrderBy != "" {
			q = q.Order(page.OrderBy)
		}
	}
	if err := q.Find(&res).Error; err != nil {
		return nil, 0, errors.Wrapf(err, "db: select %s error, query: %+v, args: %+v", b.StructName, query, args)
	}
	if total == 0 {
		total = int64(len(res))
	}
	return res, int32(total), nil
}

func Camel2Snake(s string) string {
	data := make([]byte, 0, len(s)*2)
	j := false
	num := len(s)
	for i := 0; i < num; i++ {
		d := s[i]
		// or通过ASCII码进行大小写的转化
		// 65-90（A-Z），97-122（a-z）
		// 判断如果字母为大写的A-Z就在前面拼接一个_
		if i > 0 && d >= 'A' && d <= 'Z' && j {
			data = append(data, '_')
		}
		if d != '_' {
			j = true
		}
		data = append(data, d)
	}
	// ToLower把大写字母统一转小写
	return strings.ToLower(string(data[:]))
}
