package cmd

import (
	"context"
	"errors"
	"io/ioutil"
	"skynet/db"
	"skynet/handlers"
	"skynet/utils"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage skynet user",
}

var (
	avatar     string
	userAddCmd = &cobra.Command{
		Use:   "add user [pass]",
		Short: "Add skynet user",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			connectDB(ctx)
			log.WithFields(log.Fields{
				"path": viper.GetString("database.path"),
			}).Debug("Database connected")

			content, err := ioutil.ReadFile(avatar)
			if err != nil {
				log.Fatal("Can not read file: ", err)
			}
			log.WithFields(log.Fields{
				"file": avatar,
			}).Debug("Read file success")

			var newpass string
			if len(args) == 1 {
				newpass = utils.RandString(8)
			} else {
				newpass = args[1]
			}

			newpass, err = handlers.AddUser(args[0], newpass, content)
			if err != nil {
				log.Fatal("Database error: ", err)
			}
			if len(args) == 1 {
				log.Info("New pass: ", newpass)
			}
			log.Info("Add user success")
		},
	}
)

var (
	resetall     bool
	userResetCmd = &cobra.Command{
		Use:   "reset [user]",
		Short: "Reset skynet user password",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			db.InitRedis(ctx, &db.RedisConfig{
				Address:  viper.GetString("redis.address"),
				Password: viper.GetString("redis.password"),
				DB:       viper.GetInt("redis.db"),
			})
			log.WithFields(log.Fields{
				"addr": viper.GetString("redis.address"),
			}).Debug("Redis connected")
			connectDB(ctx)
			log.WithFields(log.Fields{
				"path": viper.GetString("database.path"),
			}).Debug("Database connected")

			if !resetall {
				if len(args) != 1 {
					log.Fatal("No user specified")
				}

				newpass, err := handlers.ResetUser(args[0])
				if errors.Is(err, gorm.ErrRecordNotFound) {
					log.Fatalf("User %v not found", args[0])
				} else if err != nil {
					log.Fatal("Database error: ", err)
				}
				log.Info("New pass: ", newpass)
			} else {
				if len(args) == 1 {
					log.Fatal("Remove --all if you want to reset specific user")
				}

				newpass, err := handlers.ResetAllUser()
				if err != nil {
					log.Fatal("Database error: ", err)
				}
				if len(newpass) == 0 {
					log.Warn("No user in database")
					return
				}

				for k, v := range newpass {
					log.Infof("User %v now has new pass %v", k, v)
				}
			}
			log.Info("Reset user success")
		},
	}
)

func init() {
	userAddCmd.Flags().StringVarP(&avatar, "avatar", "a", viper.GetString("default_avatar"), "user avatar")
	userResetCmd.Flags().BoolVar(&resetall, "all", false, "reset all user password")

	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userResetCmd)
	rootCmd.AddCommand(userCmd)
}
