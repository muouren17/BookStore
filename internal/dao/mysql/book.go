package mysql

import (
	"bs/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (r *BookRepository) CreateBook(ctx context.Context, b *model.Book) (*model.Book, error) {
	book := &model.Book{
		Author:  b.Author,
		Title:   b.Title,
		ShelfID: b.ShelfID,
	}

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. 查询并锁住对应书架。
		var shelf model.Shelf
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Take(&shelf, "id = ?", b.ShelfID).Error
		if err != nil {
			return fmt.Errorf("查询书架失败: %w", normalizeError(err))
		}

		// 2. 获取锁后，再统计该书架的图书数量。
		var count int64
		err = tx.Model(&model.Book{}).
			Where("shelf_id = ?", shelf.ID).
			Count(&count).Error
		if err != nil {
			return fmt.Errorf("统计图书数量失败: %w", err)
		}

		// 3. 满容量时，结束事务，不插入图书。
		if count >= shelf.Size {
			return model.ErrShelfFull
		}

		// 4. 使用同一个 tx 插入图书。
		if err := tx.Create(book).Error; err != nil {
			return fmt.Errorf("创建图书失败: %w", err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}
	return book, nil
}

func (r *BookRepository) CountBook(ctx context.Context, shelfid int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&model.Book{}).Where("shelf_id=?", shelfid).Count(&count).Error
		if err != nil {
			return err
		}
		return nil
	})
	return count, err
}

func (r *BookRepository) DeleteBook_Shelfid(ctx context.Context, shelfid int64) error {
	book := &model.Book{}
	err := r.db.WithContext(ctx).Delete(book, "shelf_id=?", shelfid).Error
	return err
}

func (r *BookRepository) GetBook(ctx context.Context, shelfid int64, bookid int64) (m *model.Book, err error) {
	book := &model.Book{}
	err = r.db.WithContext(ctx).Take(book, "ID=? AND shelf_id=?", bookid, shelfid).Error
	if err != nil {
		return nil, normalizeError(err)
	}
	return book, nil
}

// 还需要一个书本id
func (r *BookRepository) ListBooks(ctx context.Context, shelfid int64, cursor int64, pagesize int64) ([]*model.Book, error) {
	var books []*model.Book
	err := r.db.WithContext(ctx).Where("shelf_id=? AND ID>?", shelfid, cursor).Order("id asc").Limit(int(pagesize)).Find(&books).Error
	if err != nil {
		return nil, err
	}
	return books, nil
}

func (r *BookRepository) DeleteBook(ctx context.Context, shelfid int64, bookid int64) error {
	result := r.db.WithContext(ctx).Where("shelf_id=?", shelfid).Delete(&model.Book{}, bookid)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return model.ErrNotFound
	}
	return nil
}
