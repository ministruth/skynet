package cmd

import (
	"os"

	"github.com/MXWXZ/skynet/sn"
	"github.com/MXWXZ/skynet/utils"
	"github.com/MXWXZ/skynet/utils/log"
	"github.com/vincent-petithory/dataurl"
	"github.com/ztrue/tracerr"

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
		Use:    "add $user",
		Short:  "Add skynet user",
		Args:   cobra.ExactArgs(1),
		PreRun: load,
		Run: func(cmd *cobra.Command, args []string) {
			var avatarBuf string
			var err error
			if avatar != "" {
				buf, err := os.ReadFile(avatar)
				if err != nil {
					log.NewEntry(tracerr.Wrap(err)).Fatal("Failed to read avatar")
				}
				avatarBuf = dataurl.EncodeBytes(buf)
				log.New().WithField("file", avatar).Debug("Read avatar success")
			}
			newpass := utils.RandString(8)
			if rootPerm {
				err = sn.Skynet.DB.Transaction(func(tx *gorm.DB) error {
					u, err := sn.Skynet.User.WithTx(tx).New(args[0], newpass, avatarBuf)
					if err != nil {
						return err
					}
					_, err = sn.Skynet.Group.WithTx(tx).Link([]uuid.UUID{u.ID}, []uuid.UUID{sn.Skynet.ID.Get(sn.GroupRootID)})
					return err
				})
			} else {
				_, err = sn.Skynet.User.New(args[0], newpass, avatarBuf)
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
		Use:    "reset $user",
		Short:  "Reset skynet user password",
		Args:   cobra.ExactArgs(1),
		PreRun: load,
		Run: func(cmd *cobra.Command, args []string) {
			user, err := sn.Skynet.User.GetByName(args[0])
			if err != nil {
				log.NewEntry(err).Fatal("Failed to get user")
			}
			if user == nil {
				log.New().Fatalf("User %v not found", args[0])
			}
			newpass, err := sn.Skynet.User.Reset(user.ID)
			if err != nil {
				log.NewEntry(err).Fatal("Failed to reset password")
			}
			log.New().Info("New pass: ", newpass)
			log.New().Info("Reset user success")
		},
	}
)

func init() {
	userAddCmd.Flags().StringVarP(&avatar, "avatar", "a", "", "user avatar, left empty to use default")
	userAddCmd.Flags().BoolVar(&rootPerm, "root", false, "set user to root group")

	userCmd.AddCommand(userAddCmd)
	userCmd.AddCommand(userResetCmd)
	rootCmd.AddCommand(userCmd)
}
