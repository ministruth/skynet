package cmd

import (
	"errors"
	"io/ioutil"
	"skynet/sn"
	"skynet/sn/utils"

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
	roleuser   bool
	avatar     string
	userAddCmd = &cobra.Command{
		Use:   "add user [pass]",
		Short: "Add skynet user",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			connectDB()
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

			if roleuser {
				newpass, err = sn.Skynet.User.AddUser(args[0], newpass, content, sn.RoleUser)
			} else {
				newpass, err = sn.Skynet.User.AddUser(args[0], newpass, content, sn.RoleAdmin)
				log.Warn("By default the user has admin permission, use -u/--user to force user permission")
			}

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
			connectRedis()
			log.WithFields(log.Fields{
				"addr": viper.GetString("redis.address"),
			}).Debug("Redis connected")
			connectDB()
			log.WithFields(log.Fields{
				"path": viper.GetString("database.path"),
			}).Debug("Database connected")

			if !resetall {
				if len(args) != 1 {
					log.Fatal("No user specified")
				}

				newpass, err := sn.Skynet.User.ResetUser(args[0])
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

				newpass, err := sn.Skynet.User.ResetAllUser()
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
	userAddCmd.Flags().BoolVarP(&roleuser, "user", "u", false, "set role to user permission")
	userResetCmd.Flags().BoolVar(&resetall, "all", false, "reset all user password")

	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userResetCmd)
	rootCmd.AddCommand(userCmd)
}
