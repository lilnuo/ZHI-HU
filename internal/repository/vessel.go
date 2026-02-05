package repository

import "gorm.io/gorm"

type Repositories struct {
	User     *UserRepository
	Post     *PostRepository
	Feed     *FeedRepository
	Like     *LikeRepository
	Comment  *CommentRepository
	Relation *RelationRepository
}

func NewRepositories(db *gorm.DB) *Repositories {
	return &Repositories{
		User:     NewUserRepository(db),
		Post:     NewPostRepository(db),
		Feed:     NewFeedRepository(db),
		Like:     NewLikeRepository(db),
		Comment:  NewCommentRepository(db),
		Relation: NewRelationRepository(db),
	}
}
