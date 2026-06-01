package fileupload

import (
	"database/sql"
	"fmt"

	"go-file-upload-server/domain"
)

// CreateFileShare inserts a new file_share and returns the saved record (with DB id as string).
func (p PostgresFileUploadRepository) CreateFileShare(fs domain.FileShare) (domain.FileShare, error) {
	query := `INSERT INTO file_shares (resource_type, resource_id, owner_id, grantee_id, access_level, created_at) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id`
	var id int64
	if err := p.db.QueryRow(query, fs.ResourceType, fs.ResourceID, fs.OwnerID, fs.GranteeID, fs.AccessLevel, fs.CreatedAt).Scan(&id); err != nil {
		return domain.FileShare{}, err
	}
	fs.Id = fmt.Sprintf("%d", id)
	return fs, nil
}

func (p PostgresFileUploadRepository) ListFileSharesForFile(fileID string) ([]domain.FileShare, error) {
	query := `SELECT id, COALESCE(resource_type, 'file'), COALESCE(resource_id, file_id::text), owner_id, grantee_id, access_level, created_at FROM file_shares WHERE resource_type = 'file' AND COALESCE(resource_id, file_id::text) = $1 ORDER BY created_at DESC`
	rows, err := p.db.Query(query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shares := []domain.FileShare{}
	for rows.Next() {
		var id int64
		var s domain.FileShare
		if err := rows.Scan(&id, &s.ResourceType, &s.ResourceID, &s.OwnerID, &s.GranteeID, &s.AccessLevel, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.Id = fmt.Sprintf("%d", id)
		shares = append(shares, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return shares, nil
}

func (p PostgresFileUploadRepository) ListFileSharesForUser(userID string) ([]domain.FileShare, error) {
	query := `
		SELECT 
			fs.id, 
			COALESCE(fs.resource_type, 'file') AS resource_type,
			COALESCE(fs.resource_id, fs.file_id::text) AS resource_id,
			fs.owner_id, 
			fs.grantee_id, 
			fs.access_level, 
			fs.created_at,
			COALESCE(owner.email, 'Unknown') AS owner_email,
			COALESCE(grantee.email, 'Unknown') AS grantee_email,
			CASE 
				WHEN COALESCE(fs.resource_type, 'file') = 'file' THEN COALESCE(fu.original_name, fu.file_name, 'Unknown')
				ELSE COALESCE(folder.name, 'Unknown')
			END AS resource_name,
			CASE 
				WHEN COALESCE(fs.resource_type, 'file') = 'file' THEN COALESCE(fu.file_extension, '')
				ELSE ''
			END AS resource_extension
		FROM file_shares fs
		LEFT JOIN users owner ON fs.owner_id = owner.id
		LEFT JOIN users grantee ON fs.grantee_id = grantee.id
		LEFT JOIN file_uploads fu ON COALESCE(fs.resource_type, 'file') = 'file' AND COALESCE(fs.resource_id, fs.file_id::text) = CAST(fu.id AS TEXT)
		LEFT JOIN folders folder ON COALESCE(fs.resource_type, 'file') = 'folder' AND COALESCE(fs.resource_id, fs.file_id::text) = folder.id
		WHERE fs.owner_id = $1 OR fs.grantee_id = $1 
		ORDER BY fs.created_at DESC
	`
	rows, err := p.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shares := []domain.FileShare{}
	for rows.Next() {
		var id int64
		var s domain.FileShare
		var ownerEmail string
		var granteeEmail string
		var resourceName string
		var resourceExtension string
		if err := rows.Scan(&id, &s.ResourceType, &s.ResourceID, &s.OwnerID, &s.GranteeID, &s.AccessLevel, &s.CreatedAt, &ownerEmail, &granteeEmail, &resourceName, &resourceExtension); err != nil {
			return nil, err
		}
		s.Id = fmt.Sprintf("%d", id)
		s.OwnerEmail = ownerEmail
		s.GranteeEmail = granteeEmail
		if resourceExtension != "" {
			s.ResourceName = resourceName + "." + resourceExtension
		} else {
			s.ResourceName = resourceName
		}
		shares = append(shares, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return shares, nil
}

func (p PostgresFileUploadRepository) GetFileShareByID(id string) (domain.FileShare, error) {
	var fs domain.FileShare
	query := `SELECT id, COALESCE(resource_type, 'file'), COALESCE(resource_id, file_id::text), owner_id, grantee_id, access_level, created_at FROM file_shares WHERE id = $1`
	row := p.db.QueryRow(query, id)
	var dbID int64
	if err := row.Scan(&dbID, &fs.ResourceType, &fs.ResourceID, &fs.OwnerID, &fs.GranteeID, &fs.AccessLevel, &fs.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return domain.FileShare{}, fmt.Errorf("file share not found")
		}
		return domain.FileShare{}, err
	}
	fs.Id = fmt.Sprintf("%d", dbID)
	return fs, nil
}

func (p PostgresFileUploadRepository) DeleteFileShare(id string) error {
	query := `DELETE FROM file_shares WHERE id = $1`
	_, err := p.db.Exec(query, id)
	return err
}

// UserHasAccess checks if a user has at least the given access level for a file.
// Access levels are ordered: read < write < delete. Owner implicitly has all access.
func (p PostgresFileUploadRepository) UserHasAccess(fileID string, userID string, required string) (bool, error) {
	// first check owner
	query := `SELECT owner_id, folder_id FROM file_uploads WHERE id = $1`
	row := p.db.QueryRow(query, fileID)
	var ownerID string
	var folderID sql.NullString
	if err := row.Scan(&ownerID, &folderID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if ownerID == userID {
		return true, nil
	}

	levels := map[string]int{"read": 1, "write": 2, "delete": 3}
	reqLevel := levels[required]

	// check direct file shares
	query = `SELECT access_level FROM file_shares WHERE COALESCE(resource_type, 'file') = 'file' AND COALESCE(resource_id, file_id::text) = $1 AND grantee_id = $2 ORDER BY created_at DESC LIMIT 1`
	row = p.db.QueryRow(query, fileID, userID)
	var access string
	if err := row.Scan(&access); err == nil {
		haveLevel := levels[access]
		return haveLevel >= reqLevel, nil
	} else if err != sql.ErrNoRows {
		return false, err
	}

	// check folder shares for file's folder ancestors
	if !folderID.Valid || folderID.String == "" {
		return false, nil
	}

	query = `WITH RECURSIVE ancestors AS (
		SELECT id, parent_id FROM folders WHERE id = $1
		UNION ALL
		SELECT f.id, f.parent_id FROM folders f JOIN ancestors a ON f.id = a.parent_id
	)
	SELECT access_level FROM file_shares WHERE COALESCE(resource_type, 'file') = 'folder' AND COALESCE(resource_id, file_id::text) = ANY (SELECT id FROM ancestors) AND grantee_id = $2 ORDER BY created_at DESC LIMIT 1`
	row = p.db.QueryRow(query, folderID.String, userID)
	if err := row.Scan(&access); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	haveLevel := levels[access]
	return haveLevel >= reqLevel, nil
}

func (p PostgresFileUploadRepository) UserHasFolderAccess(folderID string, userID string) (bool, error) {
	query := `SELECT owner_id FROM folders WHERE id = $1`
	row := p.db.QueryRow(query, folderID)
	var ownerID string
	if err := row.Scan(&ownerID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if ownerID == userID {
		return true, nil
	}

	query = `WITH RECURSIVE ancestors AS (
		SELECT id, parent_id FROM folders WHERE id = $1
		UNION ALL
		SELECT f.id, f.parent_id FROM folders f JOIN ancestors a ON f.id = a.parent_id
	)
	SELECT 1 FROM file_shares WHERE COALESCE(resource_type, 'file') = 'folder' AND COALESCE(resource_id, file_id::text) = ANY (SELECT id FROM ancestors) AND grantee_id = $2 LIMIT 1`
	row = p.db.QueryRow(query, folderID, userID)
	var placeholder int
	if err := row.Scan(&placeholder); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p PostgresFileUploadRepository) ListFolderContents(userID string, folderID string) ([]domain.Folder, []domain.FileUpload, error) {
	allowed, err := p.UserHasFolderAccess(folderID, userID)
	if err != nil {
		return nil, nil, err
	}
	if !allowed {
		return nil, nil, fmt.Errorf("forbidden")
	}

	var ownerID string
	if err := p.db.QueryRow(`SELECT owner_id FROM folders WHERE id = $1`, folderID).Scan(&ownerID); err != nil {
		return nil, nil, err
	}

	folders := []domain.Folder{}
	rows, err := p.db.Query(`SELECT id, owner_id, parent_id, name, created_at, updated_at FROM folders WHERE parent_id = $1 ORDER BY name`, folderID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var folder domain.Folder
		var parentID sql.NullString
		if err := rows.Scan(&folder.Id, &folder.OwnerID, &parentID, &folder.Name, &folder.CreatedAt, &folder.UpdatedAt); err != nil {
			return nil, nil, err
		}
		if parentID.Valid {
			v := parentID.String
			folder.ParentID = &v
		}
		folders = append(folders, folder)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	uploads := []domain.FileUpload{}
	tn, err := p.tableName()
	if err != nil {
		return nil, nil, err
	}
	query := fmt.Sprintf(`SELECT id, COALESCE(original_name, file_name) AS original_name, file_extension, owner_id, folder_id, size_bytes, mime_type, uploaded_at FROM %s WHERE folder_id = $1 ORDER BY uploaded_at DESC`, tn)
	rows2, err := p.db.Query(query, folderID)
	if err != nil {
		return nil, nil, err
	}
	defer rows2.Close()

	for rows2.Next() {
		var upload domain.FileUpload
		var dbID int64
		var folderID sql.NullString
		if err := rows2.Scan(&dbID, &upload.OriginalName, &upload.Extension, &upload.OwnerID, &folderID, &upload.SizeBytes, &upload.MimeType, &upload.UploadedAt); err != nil {
			return nil, nil, err
		}
		upload.Id = fmt.Sprintf("%d", dbID)
		if folderID.Valid {
			v := folderID.String
			upload.FolderID = &v
		}
		uploads = append(uploads, upload)
	}
	if err := rows2.Err(); err != nil {
		return nil, nil, err
	}

	return folders, uploads, nil
}
