package fileupload

import (
	"database/sql"
	"fmt"

	"go-file-upload-server/domain"
)

// CreateFileShare inserts a new file_share and returns the saved record (with DB id as string).
func (p PostgresFileUploadRepository) CreateFileShare(fs domain.FileShare) (domain.FileShare, error) {
	query := `INSERT INTO file_shares (file_id, owner_id, grantee_id, access_level, created_at) VALUES ($1,$2,$3,$4,$5) RETURNING id`
	var id int64
	if err := p.db.QueryRow(query, fs.FileID, fs.OwnerID, fs.GranteeID, fs.AccessLevel, fs.CreatedAt).Scan(&id); err != nil {
		return domain.FileShare{}, err
	}
	fs.Id = fmt.Sprintf("%d", id)
	return fs, nil
}

func (p PostgresFileUploadRepository) ListFileSharesForFile(fileID string) ([]domain.FileShare, error) {
	query := `SELECT id, file_id, owner_id, grantee_id, access_level, created_at FROM file_shares WHERE file_id = $1 ORDER BY created_at DESC`
	rows, err := p.db.Query(query, fileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	shares := []domain.FileShare{}
	for rows.Next() {
		var id int64
		var s domain.FileShare
		if err := rows.Scan(&id, &s.FileID, &s.OwnerID, &s.GranteeID, &s.AccessLevel, &s.CreatedAt); err != nil {
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
	// JOIN with users table to get grantee email and file_uploads to get file name
	query := `
		SELECT 
			fs.id, 
			fs.file_id, 
			fs.owner_id, 
			fs.grantee_id, 
			fs.access_level, 
			fs.created_at,
			COALESCE(u.email, 'Unknown') AS grantee_email,
			COALESCE(fu.original_name, fu.file_name, 'Unknown') AS file_name,
			COALESCE(fu.file_extension, '') AS file_extension
		FROM file_shares fs
		LEFT JOIN users u ON fs.grantee_id = u.id
		LEFT JOIN file_uploads fu ON fs.file_id = fu.id
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
		var granteeEmail string
		var fileName string
		var fileExtension string
		if err := rows.Scan(&id, &s.FileID, &s.OwnerID, &s.GranteeID, &s.AccessLevel, &s.CreatedAt, &granteeEmail, &fileName, &fileExtension); err != nil {
			return nil, err
		}
		s.Id = fmt.Sprintf("%d", id)
		s.GranteeEmail = granteeEmail
		// Reconstruct full file name with extension
		if fileExtension != "" {
			s.FileName = fileName + "." + fileExtension
		} else {
			s.FileName = fileName
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
	query := `SELECT id, file_id, owner_id, grantee_id, access_level, created_at FROM file_shares WHERE id = $1`
	row := p.db.QueryRow(query, id)
	var dbID int64
	if err := row.Scan(&dbID, &fs.FileID, &fs.OwnerID, &fs.GranteeID, &fs.AccessLevel, &fs.CreatedAt); err != nil {
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
	query := `SELECT owner_id FROM file_uploads WHERE id = $1`
	row := p.db.QueryRow(query, fileID)
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

	// check shares
	query = `SELECT access_level FROM file_shares WHERE file_id = $1 AND grantee_id = $2 ORDER BY created_at DESC LIMIT 1`
	row = p.db.QueryRow(query, fileID, userID)
	var access string
	if err := row.Scan(&access); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	levels := map[string]int{"read": 1, "write": 2, "delete": 3}
	reqLevel := levels[required]
	haveLevel := levels[access]
	return haveLevel >= reqLevel, nil
}
