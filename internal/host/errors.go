package host

import "errors"

var (
	ErrDiskAlreadyDeleted       = errors.New("disk already exists")
	ErrAttachmentAlreadyDeleted = errors.New("attachment already exists")
	ErrAttachmentNotFound       = errors.New("attachment not found")
	ErrDiskNotFound             = errors.New("disk not found")
)
