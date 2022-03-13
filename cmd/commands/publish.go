package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/usk81/easyindex"
	"github.com/usk81/easyindex/coordinator"
	"github.com/usk81/easyindex/logger"
)

var (
	publishCmd = &cobra.Command{
		Use:   "publish",
		Short: "Notifies that a URL has been updated or deleted.",
		Long:  "Notifies that a URL has been updated or deleted.",
	}

	publishUpdatedCmd = &cobra.Command{
		Use:   "update",
		Short: "Notifies that a URL has been updated.",
		Long:  "Notifies that a URL has been updated.",
		Run:   publishUpdatedCommand,
	}

	publishDeletedCmd = &cobra.Command{
		Use:   "delete",
		Short: "Notifies that a URL has been deleted.",
		Long:  "Notifies that a URL has been deleted.",
		Run:   publishDeletedCommand,
	}

	credentialsFilePath string
)

func printPublishCallResponse(
	total int,
	count int,
	skips []coordinator.SkipedPublishRequest,
) {
	fmt.Println("[result]")
	fmt.Printf("Total: %d\n", total)
	fmt.Printf("Count: %d\n", count)
	if len(skips) > 0 {
		fmt.Println("Skips:")
		for _, v := range skips {
			fmt.Printf("  [%s] %s : %s\n", v.NotificationType, v.URL, v.Reason.Error())
		}
	}
}

func publishAction(nt easyindex.NotificationType, urls []string, cf string) (err error) {
	if len(urls) == 0 {
		return
	}
	rs := make([]coordinator.PublishRequest, len(urls))
	for i, v := range urls {
		rs[i] = coordinator.PublishRequest{
			URL:              v,
			NotificationType: easyindex.NotificationTypeUpdated,
		}
	}
	l, err := logger.New("debug")
	if err != nil {
		return
	}
	s, err := coordinator.New(coordinator.Config{
		CredentialsFile: &cf,
		Logger:          l,
	})
	if err != nil {
		return
	}
	total, count, _, skips, err := s.Publish(rs)
	printPublishCallResponse(total, count, skips)
	if err != nil {
		return err
	}
	return nil
}

func publishUpdatedCommand(_ *cobra.Command, args []string) {
	if err := publishAction(easyindex.NotificationTypeUpdated, args, credentialsFilePath); err != nil {
		Exit(err, 1)
	}
}

func publishDeletedCommand(_ *cobra.Command, args []string) {
	if err := publishAction(easyindex.NotificationTypeUpdated, args, credentialsFilePath); err != nil {
		Exit(err, 1)
	}
}

func init() {
	publishUpdatedCmd.PersistentFlags().StringVarP(&credentialsFilePath, "credentials", "c", "credentials.json", "credentials file path")
	publishDeletedCmd.PersistentFlags().StringVarP(&credentialsFilePath, "credentials", "c", "credentials.json", "credentials file path")
	publishCmd.AddCommand(publishUpdatedCmd)
	publishCmd.AddCommand(publishDeletedCmd)
	RootCmd.AddCommand(publishCmd)
}
