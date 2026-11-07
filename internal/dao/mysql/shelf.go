package mysql

import (
	"bs/internal/model"
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// 数据库创建书架
func (r *ShelfRepository) CreateShelf(ctx context.Context, s *model.Shelf) (m *model.Shelf, err error) {

	shelf := &model.Shelf{
		Theme:     s.Theme,
		Size:      s.Size,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	err = r.db.WithContext(ctx).Create(shelf).Error
	if err != nil {
		log.Printf("db.WithContext(ctx).Create(shelf) failed, err:%v\n", err)
		return nil, err
	}
	return shelf, nil
}

func (r *ShelfRepository) GetShelf(ctx context.Context, shelfid int64) (m *model.Shelf, err error) {
	shelf := &model.Shelf{}
	err = r.db.WithContext(ctx).Take(shelf, "ID=?", shelfid).Error
	if err != nil {
		return nil, err
	}
	return shelf, nil
}

func (r *ShelfRepository) ListShelves(ctx context.Context) ([]*model.Shelf, error) {
	var shelves []*model.Shelf
	err := r.db.WithContext(ctx).Find(&shelves).Error
	if err != nil {
		return nil, err
	}
	return shelves, nil
}

func (r *ShelfRepository) DeleteShelf(ctx context.Context, shelfid int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		shelf := &model.Shelf{}
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Take(shelf, "id=?", shelfid).Error
		if err != nil {
			return fmt.Errorf("查询书架失败: %w", err)
		}
		err = tx.Where("shelf_id=?", shelfid).Delete(&model.Book{}).Error
		if err != nil {
			return fmt.Errorf("删除图书失败: %w", err)
		}
		if err := tx.Delete(shelf).Error; err != nil {
			return fmt.Errorf("删除书架失败: %w", err)
		}
		return nil
	})
}
