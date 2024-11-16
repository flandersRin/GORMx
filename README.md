**GORMx - A Generic ORM Extension for GORM**

[中文文档](https://github.com/flandersRin/Gormx/blob/master/doc/README.md)

GORMx is a comprehensive wrapper for GORM that provides generic repository pattern implementations and additional utility functions. It helps to simplify common database operations like CRUD, batch processing, and pagination, reducing boilerplate code. This repository is designed to boost productivity for developers who frequently use GORM in their Go projects.

#### Key Features:
- Generic Repository: Implements the repository pattern for any struct type.
- Enhanced CRUD Operations: Supports single and batch inserts, updates, and deletions by primary key or custom conditions.
- Pagination: Provides paginated query support.
- Soft Delete: Supports soft delete with a custom deletion marker.
- Flexible Query Building: Converts struct field names between camelCase and snake_case automatically.

#### Installation:
```bash
go get github.com/flandersRin/gormx
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

Check out the [documentation](docs/README.md) for more usage examples. In the `docs/` folder, include a detailed breakdown of each method, with examples in both English and Chinese.
