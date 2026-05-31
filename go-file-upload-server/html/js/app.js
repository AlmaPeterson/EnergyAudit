// ============================================================================
// Main App Controller
// ============================================================================

class FileUploadApp {
    constructor() {
        this.currentUser = null;
        this.files = [];
        this.shares = [];
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.checkAuthStatus();
    }

    setupEventListeners() {
        // Auth events
        document.getElementById('showSignup').addEventListener('click', (e) => {
            e.preventDefault();
            this.switchAuthForm('signup');
        });

        document.getElementById('showLogin').addEventListener('click', (e) => {
            e.preventDefault();
            this.switchAuthForm('login');
        });

        document.getElementById('loginBtn').addEventListener('click', () => this.handleLogin());
        document.getElementById('signupBtn').addEventListener('click', () => this.handleSignup());
        document.getElementById('logoutBtn').addEventListener('click', () => this.handleLogout());

        // File upload events
        const uploadArea = document.getElementById('uploadArea');
        const fileInput = document.getElementById('fileInput');

        uploadArea.addEventListener('click', () => fileInput.click());
        uploadArea.addEventListener('dragover', (e) => {
            e.preventDefault();
            uploadArea.classList.add('drag-over');
        });
        uploadArea.addEventListener('dragleave', () => {
            uploadArea.classList.remove('drag-over');
        });
        uploadArea.addEventListener('drop', (e) => {
            e.preventDefault();
            uploadArea.classList.remove('drag-over');
            if (e.dataTransfer.files.length) {
                this.handleFileSelect(e.dataTransfer.files[0]);
            }
        });

        fileInput.addEventListener('change', (e) => {
            if (e.target.files.length) {
                this.handleFileSelect(e.target.files[0]);
            }
        });

        // Refresh events
        document.getElementById('refreshFilesBtn').addEventListener('click', () => this.loadFiles());

        // Share events
        document.getElementById('createShareBtn').addEventListener('click', () => this.handleCreateShare());
    }

    switchAuthForm(form) {
        document.getElementById('loginForm').classList.toggle('active', form === 'login');
        document.getElementById('signupForm').classList.toggle('active', form === 'signup');
        document.getElementById('loginError').textContent = '';
        document.getElementById('signupError').textContent = '';
    }

    async handleLogin() {
        const email = document.getElementById('loginEmail').value;
        const password = document.getElementById('loginPassword').value;

        if (!email || !password) {
            UIManager.setError('loginError', 'Please fill in all fields');
            return;
        }

        try {
            const response = await APIClient.login(email, password);
            AuthManager.setToken(response.token);
            this.currentUser = { email };
            UIManager.showAppSection();
            this.loadAppData();
            UIManager.showToast('Logged in successfully');
        } catch (error) {
            UIManager.setError('loginError', error.message);
        }
    }

    async handleSignup() {
        const firstName = document.getElementById('signupFirstName').value.trim();
        const lastName = document.getElementById('signupLastName').value.trim();
        const email = document.getElementById('signupEmail').value.trim();
        const password = document.getElementById('signupPassword').value;

        if (!firstName || !lastName || !email || !password) {
            UIManager.setError('signupError', 'Please fill in all fields');
            return;
        }

        if (password.length < 6) {
            UIManager.setError('signupError', 'Password must be at least 6 characters');
            return;
        }

        try {
            await APIClient.signup(firstName, lastName, email, password);
            UIManager.showToast('Account created successfully! Please log in.');
            this.switchAuthForm('login');
            document.getElementById('loginEmail').value = email;
            document.getElementById('signupFirstName').value = '';
            document.getElementById('signupLastName').value = '';
            document.getElementById('signupEmail').value = '';
            document.getElementById('signupPassword').value = '';
        } catch (error) {
            UIManager.setError('signupError', error.message);
        }
    }

    handleLogout() {
        AuthManager.clearToken();
        this.currentUser = null;
        this.files = [];
        this.shares = [];
        UIManager.showAuthSection();
        this.switchAuthForm('login');
        document.getElementById('loginEmail').value = '';
        document.getElementById('loginPassword').value = '';
    }

    async handleFileSelect(file) {
        if (!file) return;

        // Validate file size (100MB)
        const maxSize = 100 * 1024 * 1024;
        if (file.size > maxSize) {
            UIManager.setError('uploadError', 'File size exceeds 100MB limit');
            return;
        }

        UIManager.setError('uploadError', '');
        UIManager.setLoading('uploadProgress', true);

        try {
            await APIClient.uploadFile(file);
            UIManager.showToast('File uploaded successfully');
            document.getElementById('fileInput').value = '';
            await this.loadFiles();
            await this.loadShareOptions();
        } catch (error) {
            UIManager.setError('uploadError', error.message);
        } finally {
            UIManager.setLoading('uploadProgress', false);
        }
    }

    async loadAppData() {
        document.getElementById('userDisplay').textContent = 
            `Logged in as ${this.currentUser.email}`;
        
        await this.loadFiles();
        await this.loadShares();
        await this.loadShareOptions();
    }

    async loadFiles() {
        UIManager.setLoading('filesLoading', true);
        document.getElementById('filesList').classList.add('hidden');
        document.getElementById('filesEmpty').classList.add('hidden');

        try {
            this.files = await APIClient.getUploads() || [];
            
            if (this.files.length === 0) {
                document.getElementById('filesEmpty').classList.remove('hidden');
            } else {
                this.renderFilesList();
                document.getElementById('filesList').classList.remove('hidden');
            }
        } catch (error) {
            UIManager.showToast('Failed to load files', 'error');
        } finally {
            UIManager.setLoading('filesLoading', false);
        }
    }

    renderFilesList() {
        const list = document.getElementById('filesList');
        list.innerHTML = '';

        this.files.forEach(file => {
            const fileName = `${file.originalName}.${file.extension}`;
            const uploadDate = new Date(file.uploadedAt).toLocaleString();
            
            const item = document.createElement('div');
            item.className = 'file-item';
            item.innerHTML = `
                <div class="file-info">
                    <div class="file-name">${UIManager.escapeHtml(fileName)}</div>
                    <div class="file-meta">
                        Uploaded: ${uploadDate}
                    </div>
                </div>
                <div class="file-actions">
                    <button class="btn btn-small" data-download-id="${file.id}" data-file-name="${UIManager.escapeHtml(fileName)}">
                        Download
                    </button>
                    <button class="btn btn-small btn-danger" data-file-id="${file.id}">
                        Delete
                    </button>
                </div>
            `;

            item.querySelector('[data-download-id]').addEventListener('click', (e) => {
                const button = e.target;
                this.handleDownloadFile(button.dataset.downloadId, button.dataset.fileName);
            });
            item.querySelector('[data-file-id]').addEventListener('click', (e) => {
                this.handleDeleteFile(e.target.dataset.fileId);
            });

            list.appendChild(item);
        });
    }

    async handleDownloadFile(fileId, fileName) {
        try {
            const blob = await APIClient.downloadUpload(fileId);
            const url = URL.createObjectURL(blob);
            const anchor = document.createElement('a');
            anchor.href = url;
            anchor.download = fileName;
            document.body.appendChild(anchor);
            anchor.click();
            anchor.remove();
            URL.revokeObjectURL(url);
        } catch (error) {
            UIManager.showToast(error.message, 'error');
        }
    }

    async handleDeleteFile(fileId) {
        if (!confirm('Are you sure you want to delete this file?')) return;

        try {
            await APIClient.deleteUpload(fileId);
            UIManager.showToast('File deleted successfully');
            await this.loadFiles();
            await this.loadShareOptions();
        } catch (error) {
            UIManager.showToast(error.message, 'error');
        }
    }

    async loadShareOptions() {
        const select = document.getElementById('shareFileSelect');
        select.innerHTML = '<option value="">Choose a file...</option>';

        this.files.forEach(file => {
            const fileName = `${file.originalName}.${file.extension}`;
            const option = document.createElement('option');
            option.value = file.id;
            option.textContent = fileName;
            select.appendChild(option);
        });
    }

    async handleCreateShare() {
        const fileId = document.getElementById('shareFileSelect').value;
        const granteeEmail = document.getElementById('shareGranteeId').value.trim();
        const accessLevel = document.getElementById('shareAccessLevel').value;

        UIManager.setError('shareError', '');

        if (!fileId || !granteeEmail) {
            UIManager.setError('shareError', 'Please select a file and enter recipient email');
            return;
        }

        try {
            await APIClient.createShare(fileId, granteeEmail, accessLevel);
            UIManager.showToast('File shared successfully');
            document.getElementById('shareFileSelect').value = '';
            document.getElementById('shareGranteeId').value = '';
            await this.loadShares();
        } catch (error) {
            UIManager.setError('shareError', error.message);
        }
    }

    async loadShares() {
        UIManager.setLoading('sharesLoading', true);
        document.getElementById('sharesList').classList.add('hidden');
        document.getElementById('sharesEmpty').classList.add('hidden');

        try {
            this.shares = await APIClient.getShares() || [];
            
            if (this.shares.length === 0) {
                document.getElementById('sharesEmpty').classList.remove('hidden');
            } else {
                this.renderSharesList();
                document.getElementById('sharesList').classList.remove('hidden');
            }
        } catch (error) {
            UIManager.showToast('Failed to load shares', 'error');
        } finally {
            UIManager.setLoading('sharesLoading', false);
        }
    }

    renderSharesList() {
        const list = document.getElementById('sharesList');
        list.innerHTML = '';

        if (this.shares.length === 0) return;

        const table = document.createElement('table');
        table.className = 'shares-table-content';
        table.innerHTML = `
            <thead>
                <tr>
                    <th>File</th>
                    <th>Shared With / By</th>
                    <th>Access Level</th>
                    <th>Created</th>
                    <th>Action</th>
                </tr>
            </thead>
            <tbody></tbody>
        `;

        const tbody = table.querySelector('tbody');
        this.shares.forEach(share => {
            const createdDate = new Date(share.createdAt).toLocaleString();
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${UIManager.escapeHtml(share.fileName || 'Unknown')}</td>
                <td>${UIManager.escapeHtml(share.email || 'Unknown')}</td>
                <td>${UIManager.escapeHtml(share.accessLevel || 'view')}</td>
                <td>${createdDate}</td>
                <td>
                    <button class="btn btn-small btn-danger" data-share-id="${share.id}">
                        Remove
                    </button>
                </td>
            `;

            row.querySelector('[data-share-id]').addEventListener('click', (e) => {
                this.handleDeleteShare(e.target.dataset.shareId);
            });

            tbody.appendChild(row);
        });

        list.appendChild(table);
    }

    async handleDeleteShare(shareId) {
        if (!confirm('Are you sure you want to remove this share?')) return;

        try {
            await APIClient.deleteShare(shareId);
            UIManager.showToast('Share removed successfully');
            await this.loadShares();
        } catch (error) {
            UIManager.showToast(error.message, 'error');
        }
    }

    checkAuthStatus() {
        if (AuthManager.isAuthenticated()) {
            this.currentUser = { email: 'User' };
            UIManager.showAppSection();
            this.loadAppData();
        } else {
            UIManager.showAuthSection();
        }
    }
}

// ============================================================================
// Initialize App
// ============================================================================

document.addEventListener('DOMContentLoaded', () => {
    new FileUploadApp();
});
