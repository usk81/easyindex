package easyindex

type NotificationType string

// The URL life cycle event that Google is being notified about.
const (
	// Unspecified ...
	NotificationTypeUnspecified = NotificationType("URL_NOTIFICATION_TYPE_UNSPECIFIED")
	// Updated means that The given URL (Web document) has been updated.
	NotificationTypeUpdated = NotificationType("URL_UPDATED")
	// Deleted means that The given URL (Web document) has been deleted.
	NotificationTypeDeleted = NotificationType("URL_DELETED")
)
