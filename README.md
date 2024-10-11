**GORMx - A Generic ORM Extension for GORM**

[中文文档]()

GORMx is a comprehensive wrapper for GORM that provides generic repository pattern implementations and additional utility functions. It helps to simplify common database operations like CRUD, batch processing, and pagination, reducing boilerplate code. This repository is designed to boost productivity for developers who frequently use GORM in their Go projects.

#### Key Features:
- Generic Repository: Implements the repository pattern for any struct type.
- Enhanced CRUD Operations: Supports single and batch inserts, updates, and deletions by primary key or custom conditions.
- Pagination: Provides paginated query support.
- Soft Delete: Supports soft delete with a custom deletion marker.
- Flexible Query Building: Converts struct field names between camelCase and snake_case automatically.

#### Installation:
```bash
go get github.com/<your-username>/gormx
```

#### Usage:
1. **Basic CRUD**:
   ```go
   repo := NewBaseRepo[YourStruct](data)
   repo.Insert(ctx, &yourStruct)
   repo.SelectOneByPK(ctx, primaryKey)
   ```

2. **Batch Insert**:
   ```go
   repo.BatchInsert(ctx, []*YourStruct{&obj1, &obj2}, 100)
   ```

3. **Delete by Primary Key**:
   ```go
   repo.DeleteByPK(ctx, primaryKey)
   ```

Check out the [documentation](docs/README.md) for more usage examples.

### 中文

**GORMx - GORM 泛型封装扩展库**

GORMx 是一个 GORM 的封装库，提供了通用的仓储模式实现和其他实用功能。通过减少样板代码，简化了常见的数据库操作，如增删改查、批量处理和分页查询。此仓库旨在提高使用 GORM 开发 Go 项目的开发者的生产力。

#### 主要功能:
- 通用仓储模式: 为任意结构体类型实现仓储模式。
- 增强的 CRUD 操作: 支持通过主键或自定义条件进行单个或批量的插入、更新和删除。
- 分页查询: 提供分页查询支持。
- 软删除: 支持带有自定义删除标记的软删除功能。
- 灵活的查询构建: 自动在驼峰命名和下划线命名之间转换字段名。

#### 安装:
```bash
go get github.com/<your-username>/gormx
```

#### 使用示例:
1. **基础 CRUD**:
   ```go
   repo := NewBaseRepo[YourStruct](data)
   repo.Insert(ctx, &yourStruct)
   repo.SelectOneByPK(ctx, primaryKey)
   ```

2. **批量插入**:
   ```go
   repo.BatchInsert(ctx, []*YourStruct{&obj1, &obj2}, 100)
   ```

3. **通过主键删除**:
   ```go
   repo.DeleteByPK(ctx, primaryKey)
   ```

查看更多使用示例，请参阅[文档](docs/README.md)。

---

### Documentation (中英文)

In the `docs/` folder, include a detailed breakdown of each method, with examples in both English and Chinese.