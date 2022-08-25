package cmd

import (
	"io/ioutil"
	"skynet/db"
	"skynet/handler"
	"skynet/utils/log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/spf13/cobra"
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
			db.NewDB()
			handler.Init()

			avatarBuf, err := ioutil.ReadFile(avatar)
			if err != nil {
				log.NewEntry(err).Fatal("Failed to read avatar")
			}
			log.New().WithField("file", avatar).Debug("Read avatar success")

			var newpass string
			if rootPerm {
				err = db.DB.Transaction(func(tx *gorm.DB) error {
					var u *db.User
					u, newpass, err = handler.User.WithTx(tx).New(args[0], "", avatarBuf)
					if err != nil {
						return err
					}
					_, err = handler.Group.WithTx(tx).Link([]uuid.UUID{u.ID}, []uuid.UUID{db.GetDefaultID(db.GroupRootID)})
					return err
				})
			} else {
				_, newpass, err = handler.User.New(args[0], "", avatarBuf)
				log.New().Warn("By default the user has no permission, use --root to add to root group")
			}

			if err != nil {
				log.NewEntry(err).Fatal("Failed to create user")
			}
			log.New().Info("New pass: ", newpass)
			log.New().Info("Add user success")
		},
	}
)

var (
	userResetCmd = &cobra.Command{
		Use:   "reset $user",
		Short: "Reset skynet user password",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			db.NewRedis()
			db.NewDB()
			handler.Init()

			user, err := handler.User.GetByName(args[0])
			if err != nil {
				log.NewEntry(err).Fatal("Failed to get user")
			}
			if user == nil {
				log.New().Fatalf("User %v not found", args[0])
			}
			newpass, err := handler.User.Reset(user.ID)
			if err != nil {
				log.NewEntry(err).Fatal("Failed to reset password")
			}
			log.New().Info("New pass: ", newpass)
			log.New().Info("Reset user success")
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
