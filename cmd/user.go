package cmd

import (
	"io/ioutil"
	"skynet/handler"
	"skynet/sn"
	"skynet/sn/impl"
	"skynet/sn/utils"

	"github.com/google/uuid"
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
	rootPerm   bool
	avatar     string
	userAddCmd = &cobra.Command{
		Use:   "add $user",
		Short: "Add skynet user",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			impl.ConnectDB()
			log.WithField("path", viper.GetString("database.path")).Debug("Database connected")
			sn.Skynet.User = handler.NewUser()
			sn.Skynet.Group = handler.NewGroup()

			content, err := ioutil.ReadFile(avatar)
			if err != nil {
				utils.WithTrace(err).Fatal(err)
			}
			webp, err := utils.ConvertWebp(content)
			if err != nil {
				utils.WithTrace(err).Fatal(err)
			}
			log.WithField("file", avatar).Debug("Read file success")

			var newpass string
			if rootPerm {
				err = sn.Skynet.GetDB().Transaction(func(tx *gorm.DB) error {
					var u *sn.User
					u, newpass, err = sn.Skynet.User.WithTx(tx).New(args[0], "", webp)
					if err != nil {
						return err
					}
					_, err = sn.Skynet.Group.Link([]uuid.UUID{u.ID}, []uuid.UUID{sn.Skynet.GetID(sn.GroupRootID)})
					return err
				})
			} else {
				_, newpass, err = sn.Skynet.User.New(args[0], "", webp)
				log.Warn("By default the user has no permission, use --root to add to root group")
			}

			if err != nil {
				utils.WithTrace(err).Fatal(err)
			}
			log.Info("New pass: ", newpass)
			log.Info("Add user success")
		},
	}
)

var (
	userResetCmd = &cobra.Command{
		Use:   "reset $user",
		Short: "Reset skynet user password",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			impl.ConnectRedis()
			log.WithField("addr", viper.GetString("redis.address")).Debug("Redis connected")
			impl.ConnectDB()
			log.WithField("path", viper.GetString("database.path")).Debug("Database connected")
			sn.Skynet.User = handler.NewUser()

			user, err := sn.Skynet.User.GetByName(args[0])
			if err != nil {
				utils.WithTrace(err).Fatal(err)
			}
			if user == nil {
				log.Fatalf("User %v not found", args[0])
			}
			newpass, err := sn.Skynet.User.Reset(user.ID)
			if err != nil {
				utils.WithTrace(err).Fatal(err)
			}
			log.Info("New pass: ", newpass)
			log.Info("Reset user success")
		},
	}
)

func init() {
	userAddCmd.Flags().StringVarP(&avatar, "avatar", "a", "default.webp", "user avatar")
	userAddCmd.Flags().BoolVar(&rootPerm, "root", false, "set user to root group")

	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userResetCmd)
	rootCmd.AddCommand(userCmd)
}
