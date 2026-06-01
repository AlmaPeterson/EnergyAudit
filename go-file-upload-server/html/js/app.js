// ============================================================================
// Main App Controller
// ============================================================================

class EnergyAuditApp {
    constructor() {
        this.currentUser = null;
        this.currentJob = null;
        this.currentTask = null;
        this.jobs = [];
        this.tasks = [];
        this.timeEntries = [];
        this.audits = [];
        this.images = [];
        this.init();
    }

    init() {
        this.setupEventListeners();
        this.checkAuthStatus();
    }

    setupEventListeners() {
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

        document.getElementById('createJobBtn').addEventListener('click', () => this.handleCreateJob());
        document.getElementById('refreshJobsBtn').addEventListener('click', () => this.loadJobs());
        document.getElementById('createTaskBtn').addEventListener('click', () => this.handleCreateTask());
        document.getElementById('refreshTasksBtn').addEventListener('click', () => this.loadTasks());
        document.getElementById('createTimeEntryBtn').addEventListener('click', () => this.handleCreateTimeEntry());
        document.getElementById('stopTimeEntryBtn').addEventListener('click', () => this.handleStopActiveTimeEntry());
        document.getElementById('createAuditBtn').addEventListener('click', () => this.handleCreateAudit());
        document.getElementById('uploadImageBtn').addEventListener('click', () => this.handleUploadImage());

        ['loginEmail', 'loginPassword'].forEach((id) => {
            const input = document.getElementById(id);
            if (!input) return;
            input.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    this.handleLogin();
                }
            });
        });
    }

    switchAuthForm(form) {
        document.getElementById('loginForm').classList.toggle('active', form === 'login');
        document.getElementById('signupForm').classList.toggle('active', form === 'signup');
        UIManager.setError('loginError', '');
        UIManager.setError('signupError', '');
    }

    async handleLogin() {
        const email = document.getElementById('loginEmail').value.trim();
        const password = document.getElementById('loginPassword').value;

        if (!email || !password) {
            UIManager.setError('loginError', 'Please fill in all fields');
            return;
        }

        try {
            const response = await APIClient.login(email, password);
            AuthManager.setToken(response.token);
            this.currentUser = {
                id: AuthManager.getUserId(),
                email: AuthManager.getUserEmail() || email,
            };
            UIManager.showAppSection();
            await this.loadDashboard();
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
            UIManager.showToast('Account created successfully. Please log in.');
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
        this.currentJob = null;
        this.currentTask = null;
        UIManager.showAuthSection();
        this.switchAuthForm('login');
    }

    async loadDashboard() {
        document.getElementById('userDisplay').textContent = `Logged in as ${this.currentUser?.email || 'User'}`;
        await this.loadJobs();
        this.clearJobSelection();
        this.clearTaskSelection();
    }

    async loadJobs() {
        try {
            this.jobs = await APIClient.getJobs() || [];
            this.renderJobsList();
        } catch (error) {
            UIManager.showToast(error.message, 'error');
        }
    }

    async loadTasks() {
        if (!this.currentJob) {
            this.tasks = [];
            this.renderTasksList();
            return;
        }

        try {
            this.tasks = await APIClient.listTasks(this.currentJob.id) || [];
            this.renderTasksList();
        } catch (error) {
            UIManager.showToast(error.message, 'error');
        }
    }

    async loadTimeEntries() {
        if (!this.currentTask) {
            this.timeEntries = [];
            this.renderTimeEntries();
            return;
        }

        try {
            this.timeEntries = await APIClient.listTimeEntries(this.currentTask.id) || [];
            this.renderTimeEntries();
        } catch (error) {
            UIManager.showToast(error.message, 'error');
        }
    }

    async loadAudits() {
        if (!this.currentTask) {
            this.audits = [];
            this.renderAudits();
            return;
        }

        UIManager.setLoading('auditStatus', true);
        try {
            this.audits = await APIClient.listEnergyAudits(this.currentTask.id) || [];
            this.renderAudits();
        } catch (error) {
            UIManager.showToast(error.message, 'error');
        } finally {
            UIManager.setLoading('auditStatus', false);
        }
    }

    async loadImages() {
        if (!this.currentTask) {
            this.images = [];
            this.renderImages();
            return;
        }

        try {
            this.images = await APIClient.listImages(this.currentTask.id) || [];
            this.renderImages();
        } catch (error) {
            UIManager.showToast(error.message, 'error');
        }
    }

    async handleCreateJob() {
        const title = document.getElementById('jobTitleInput').value.trim();
        const description = document.getElementById('jobDescriptionInput').value.trim();
        UIManager.setError('jobError', '');

        if (!title) {
            UIManager.setError('jobError', 'Job title is required');
            return;
        }

        try {
            await APIClient.createJob(title, description);
            UIManager.showToast('Job created successfully');
            document.getElementById('jobTitleInput').value = '';
            document.getElementById('jobDescriptionInput').value = '';
            await this.loadJobs();
        } catch (error) {
            UIManager.setError('jobError', error.message);
        }
    }

    async handleCreateTask() {
        const title = document.getElementById('taskTitleInput').value.trim();
        const description = document.getElementById('taskDescriptionInput').value.trim();
        UIManager.setError('taskError', '');

        if (!this.currentJob) {
            UIManager.setError('taskError', 'Select a job before adding a task');
            return;
        }

        if (!title) {
            UIManager.setError('taskError', 'Task title is required');
            return;
        }

        try {
            await APIClient.createTask(this.currentJob.id, title, description);
            UIManager.showToast('Task added successfully');
            document.getElementById('taskTitleInput').value = '';
            document.getElementById('taskDescriptionInput').value = '';
            await this.loadTasks();
        } catch (error) {
            UIManager.setError('taskError', error.message);
        }
    }

    async handleCreateTimeEntry() {
        const start = document.getElementById('timeEntryStartInput').value;
        const end = document.getElementById('timeEntryEndInput').value;
        const note = document.getElementById('timeEntryNoteInput').value.trim();
        UIManager.setError('timeEntryError', '');

        if (!this.currentTask) {
            UIManager.setError('timeEntryError', 'Select a task before logging time');
            return;
        }

        if (!start) {
            UIManager.setError('timeEntryError', 'Start time is required');
            return;
        }

        try {
            await APIClient.createTimeEntry(this.currentTask.id, start, end || null, note);
            UIManager.showToast('Time entry saved');
            document.getElementById('timeEntryStartInput').value = '';
            document.getElementById('timeEntryEndInput').value = '';
            document.getElementById('timeEntryNoteInput').value = '';
            await this.loadTimeEntries();
        } catch (error) {
            UIManager.setError('timeEntryError', error.message);
        }
    }

    async handleStopActiveTimeEntry() {
        if (!this.currentTask) {
            UIManager.showToast('Select a task before stopping a timer', 'error');
            return;
        }

        const activeEntry = this.timeEntries.find((entry) => !entry.endTime);
        if (!activeEntry) {
            UIManager.showToast('No active timer found', 'error');
            return;
        }

        try {
            await APIClient.stopTimeEntry(activeEntry.id);
            UIManager.showToast('Timer stopped');
            await this.loadTimeEntries();
        } catch (error) {
            UIManager.showToast(error.message, 'error');
        }
    }

    async handleCreateAudit() {
        const easy = document.getElementById('auditEasy').checked;
        const hard = document.getElementById('auditHard').checked;
        const fun = document.getElementById('auditFun').checked;
        const notFun = document.getElementById('auditNotFun').checked;
        const efficiencyRating = parseInt(document.getElementById('auditEfficiencyRating').value, 10);
        const notes = document.getElementById('auditNotes').value.trim();
        UIManager.setError('auditError', '');

        if (!this.currentTask || !this.currentJob) {
            UIManager.setError('auditError', 'Select a job and task before saving an audit');
            return;
        }

        if (Number.isNaN(efficiencyRating) || efficiencyRating < 1 || efficiencyRating > 10) {
            UIManager.setError('auditError', 'Efficiency rating must be between 1 and 10');
            return;
        }

        try {
            await APIClient.createEnergyAudit(this.currentJob.id, this.currentTask.id, easy, hard, fun, notFun, efficiencyRating, notes);
            UIManager.showToast('Audit saved');
            document.getElementById('auditEasy').checked = false;
            document.getElementById('auditHard').checked = false;
            document.getElementById('auditFun').checked = false;
            document.getElementById('auditNotFun').checked = false;
            document.getElementById('auditEfficiencyRating').value = '';
            document.getElementById('auditNotes').value = '';
            await this.loadAudits();
        } catch (error) {
            UIManager.setError('auditError', error.message);
        }
    }

    async handleUploadImage() {
        const input = document.getElementById('imageUploadInput');
        const photoType = document.getElementById('imagePhotoType').value;
        UIManager.setError('imageError', '');

        if (!this.currentTask) {
            UIManager.setError('imageError', 'Select a task before uploading a photo');
            return;
        }

        if (!input.files.length) {
            UIManager.setError('imageError', 'Select an image to upload');
            return;
        }

        const file = input.files[0];
        try {
            await APIClient.uploadImage(file, this.currentTask.id, this.currentJob?.id, photoType);
            UIManager.showToast('Photo uploaded successfully');
            input.value = '';
            await this.loadImages();
        } catch (error) {
            UIManager.setError('imageError', error.message);
        }
    }

    async renderJobsList() {
        const list = document.getElementById('jobsList');
        list.innerHTML = '';

        if (this.jobs.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'No jobs yet. Create one to get started.';
            list.appendChild(empty);
            return;
        }

        this.jobs.forEach((job) => {
            const card = document.createElement('div');
            card.className = 'list-card';
            card.innerHTML = `
                <h3>${UIManager.escapeHtml(job.title)}</h3>
                <p>${UIManager.escapeHtml(job.description || 'No description added')}</p>
                <button class="btn btn-small" data-job-id="${job.id}">${this.currentJob?.id === job.id ? 'Selected' : 'Select'}</button>
            `;

            card.querySelector('button').addEventListener('click', () => this.selectJob(job));
            list.appendChild(card);
        });
    }

    async renderTasksList() {
        const list = document.getElementById('tasksList');
        list.innerHTML = '';

        if (!this.currentJob) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'Choose a job to display tasks.';
            list.appendChild(empty);
            return;
        }

        if (this.tasks.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'No tasks for this job yet.';
            list.appendChild(empty);
            return;
        }

        this.tasks.forEach((task) => {
            const card = document.createElement('div');
            card.className = 'list-card';
            card.innerHTML = `
                <h3>${UIManager.escapeHtml(task.title)}</h3>
                <p>${UIManager.escapeHtml(task.description || 'No description added')}</p>
                <button class="btn btn-small" data-task-id="${task.id}">${this.currentTask?.id === task.id ? 'Selected' : 'Select'}</button>
            `;

            card.querySelector('button').addEventListener('click', () => this.selectTask(task));
            list.appendChild(card);
        });
    }

    renderTimeEntries() {
        const list = document.getElementById('timeEntriesList');
        list.innerHTML = '';

        if (!this.currentTask) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'Select a task to see time entries.';
            list.appendChild(empty);
            return;
        }

        if (this.timeEntries.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'No entries logged yet.';
            list.appendChild(empty);
            return;
        }

        this.timeEntries.forEach((entry) => {
            const card = document.createElement('div');
            card.className = 'list-card';
            card.innerHTML = `
                <h3>${entry.note ? UIManager.escapeHtml(entry.note) : 'Time entry'}</h3>
                <p>${UIManager.escapeHtml(new Date(entry.startTime).toLocaleString())} - ${entry.endTime ? UIManager.escapeHtml(new Date(entry.endTime).toLocaleString()) : 'In progress'}</p>
            `;
            list.appendChild(card);
        });
    }

    renderAudits() {
        const list = document.getElementById('auditList');
        list.innerHTML = '';

        if (!this.currentTask) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'Select a task to show audit status.';
            list.appendChild(empty);
            return;
        }

        if (this.audits.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'No audits have been logged for this task yet.';
            list.appendChild(empty);
            return;
        }

        this.audits.forEach((audit) => {
            const card = document.createElement('div');
            card.className = 'list-card';
            card.innerHTML = `
                <h3>Audit at ${UIManager.escapeHtml(new Date(audit.createdAt).toLocaleString())}</h3>
                <p>Easy: ${audit.easy ? 'Yes' : 'No'} · Hard: ${audit.hard ? 'Yes' : 'No'} · Fun: ${audit.fun ? 'Yes' : 'No'} · Not Fun: ${audit.notFun ? 'Yes' : 'No'}</p>
                <p>Efficiency: ${UIManager.escapeHtml(String(audit.efficiencyRating))}</p>
                <p>${UIManager.escapeHtml(audit.notes || 'No notes')}</p>
            `;
            list.appendChild(card);
        });
    }

    renderImages() {
        const list = document.getElementById('imagesList');
        list.innerHTML = '';

        if (!this.currentTask) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'Select a task to view uploaded photos.';
            list.appendChild(empty);
            return;
        }

        if (this.images.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'No photos uploaded yet.';
            list.appendChild(empty);
            return;
        }

        this.images.forEach((image) => {
            const card = document.createElement('div');
            card.className = 'list-card';
            const createdAt = new Date(image.createdAt).toLocaleString();
            card.innerHTML = `
                <h3>${UIManager.escapeHtml(image.photoType || 'Photo')}</h3>
                <p>${UIManager.escapeHtml(image.originalName || image.fileName || 'Uploaded image')}</p>
                <p>${UIManager.escapeHtml(createdAt)}</p>
                <button class="btn btn-small" data-image-id="${image.id}">Download</button>
            `;
            card.querySelector('button').addEventListener('click', () => this.downloadImage(image.id, image.originalName || 'photo')); 
            list.appendChild(card);
        });
    }

    async downloadImage(imageId, fileName) {
        try {
            const blob = await APIClient.downloadImage(imageId);
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

    async selectJob(job) {
        this.currentJob = job;
        document.getElementById('selectedJobTitle').textContent = `Selected Job: ${job.title}`;
        await this.loadTasks();
        this.clearTaskSelection();
    }

    async selectTask(task) {
        this.currentTask = task;
        document.getElementById('selectedTaskTitle').textContent = `Selected Task: ${task.title}`;
        await Promise.all([this.loadTimeEntries(), this.loadAudits(), this.loadImages()]);
    }

    clearJobSelection() {
        this.currentJob = null;
        document.getElementById('selectedJobTitle').textContent = 'Select a job to manage tasks.';
        this.tasks = [];
        this.renderTasksList();
    }

    clearTaskSelection() {
        this.currentTask = null;
        document.getElementById('selectedTaskTitle').textContent = 'Select a task to log time.';
        this.timeEntries = [];
        this.audits = [];
        this.images = [];
        this.renderTimeEntries();
        this.renderAudits();
        this.renderImages();
    }

    checkAuthStatus() {
        if (AuthManager.isAuthenticated()) {
            this.currentUser = {
                email: AuthManager.getUserEmail() || 'User',
                id: AuthManager.getUserId(),
            };
            UIManager.showAppSection();
            this.loadDashboard();
        } else {
            UIManager.showAuthSection();
        }
    }
}

// ============================================================================
// Initialize App
// ============================================================================

document.addEventListener('DOMContentLoaded', () => {
    new EnergyAuditApp();
});
