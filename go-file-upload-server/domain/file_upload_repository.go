package domain

type FileUploadRepository interface {
	UploadFile(fileUpload FileUpload, fileData []byte) (FileUpload, error)
	GetFileUploadByID(id string) (FileUpload, error)
	GetFileUploadDataByID(id string) ([]byte, error)
	ListFileUploads() ([]FileUpload, error)
	ListFileUploadsByFolder(ownerID string, folderID *string) ([]FileUpload, error)
	DeleteFileUpload(id string) error

	CreateFolder(folder Folder) (Folder, error)
	ListFoldersForUser(userID string, parentID *string) ([]Folder, error)
	GetFolderByID(id string) (Folder, error)
	DeleteFolder(id string) error

	UserHasFolderAccess(folderID string, userID string) (bool, error)
	ListFolderContents(userID string, folderID string) ([]Folder, []FileUpload, error)
}
