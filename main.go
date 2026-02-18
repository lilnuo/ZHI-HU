package main

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"go-zhihu/cmd/start"
	"go-zhihu/config"
	_ "go-zhihu/docs"
	"go-zhihu/internal/handler"
	"go-zhihu/internal/middleware"
	"go-zhihu/internal/model"
	"go-zhihu/internal/repository"
	"go-zhihu/internal/service"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
)

//@title Go-Zhihu API
//@version 1.0
//@description 这是模仿知乎的一个api服务
//@termsOfService http://swagger.io/terms/
//@contact.name API Support
//@contact.eamil support@swagger.io
//@license.name Apache 2.0
//@license.url http://www.apache.org/license/LICENSE-2.0.html
//@host localhost:8080
//@BasePath /api/v1

func main() {
	if err := config.Init("config/config.yaml"); err != nil {
		log.Fatalf("Config start failed:%v", err)
	}
	gin.SetMode(config.Setting.Server.Mode)
	dsn := config.Setting.Database.GetDSN()
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		Logger:                                   logger.Default.LogMode(logger.Info),
		DisableAutomaticPing:                     true,
	})
	if err != nil {
		log.Fatalf("Mysql start failed:%v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}
	defer func(sqlDB *sql.DB) {
		err := sqlDB.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(sqlDB)
	db.Exec("SET FOREIGN_KEY_CHECKS = 0")
	tables := []interface{}{
		&model.Notification{},
		&model.Like{},
		&model.User{},
		&model.Post{},
		&model.Relation{},
		&model.Comment{},
	}
	for _, table := range tables {
		err := db.Migrator().DropTable(table)
		if err != nil {
			return
		}
	}
	err = db.AutoMigrate(
		&model.Notification{},
		&model.Like{},
		&model.Comment{},
		&model.Post{},
		&model.Connection{},
		&model.User{},
		&model.Relation{})
	db.Exec("SET FOREIGN_KEY_CHECKS = 1")
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Setting.Redis.GetAddr(),
		Password: config.Setting.Redis.Password,
		DB:       config.Setting.Redis.DB,
	})
	repos := repository.NewRepositories(db)
	jwtSecret := config.Setting.JWT.Secret
	socialService := service.NewService(
		db,
		rdb,
		repos,
		jwtSecret,
	)
	httpHandler := handler.NewHandler(socialService, db)
	r := gin.Default()
	err = r.SetTrustedProxies(nil)
	if err != nil {
		return
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.Use(middleware.CustomRecovery())
	r.Use(middleware.RateLimit(rdb, 20))
	r.Use(middleware.CheckStatus(repos.User))
	start.SetRoute(r, httpHandler)

}
