package service

import "admin/internal/filex"

type fileStorageService = filex.Storage
type storedFileObject = filex.StoredFileObject

var newFileStorageService = filex.NewStorage

var (
	defaultPresignExpire = filex.DefaultPresignExpire
	maxInlineDownload    = filex.MaxInlineDownload
	normalizeContentType = filex.NormalizeContentType
	normalizeBucketName  = filex.NormalizeBucketName
	buildStoredObjectName = filex.BuildStoredObjectName
	randomSuffix         = filex.RandomSuffix
	humanReadableSize    = filex.HumanReadableSize
	fileMimeFromRecord   = filex.FileMimeFromRecord
	buildDownloadDisposition = filex.BuildDownloadDisposition
	buildFileDirectory   = filex.BuildFileDirectory
	asStorageObject      = filex.AsStorageObject
	joinStoredObjectPath = filex.JoinStoredObjectPath
	fileStringPtr        = filex.StringPtr
)
