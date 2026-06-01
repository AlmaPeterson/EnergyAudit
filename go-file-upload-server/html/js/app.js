// ============================================================================
// Main App Controller
// ============================================================================

class EnergyAuditApp {
    constructor() {
        this.currentUser = null;
        this.currentJob = null;
        this.currentTask = null;
        this.currentPage = 'jobs';
        this.currentTaskPage = 'time';
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
        document.getElementById('startTimerBtn').addEventListener('click', () => this.handleStartTimer());
        document.getElementById('stopTimeEntryBtn').addEventListener('click', () => this.handleStopActiveTimeEntry());
        document.getElementById('createAuditBtn').addEventListener('click', () => this.handleCreateAudit());
        document.getElementById('uploadImageBtn').addEventListener('click', () => this.handleUploadImage());
        document.getElementById('backToJobsBtn').addEventListener('click', () => this.showPage('jobs'));
        document.getElementById('backToTasksBtn').addEventListener('click', () => this.showPage('tasks'));
        document.getElementById('showTimePageBtn').addEventListener('click', () => this.showTaskSubpage('time'));
        document.getElementById('showAuditPageBtn').addEventListener('click', () => this.showTaskSubpage('audit'));
        document.getElementById('showPhotoPageBtn').addEventListener('click', () => this.showTaskSubpage('photo'));

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

        this.addEnterKeyHandlers();
    }

    addEnterKeyHandlers() {
        const enterTargets = [
            { id: 'jobTitleInput', action: () => this.handleCreateJob() },
            { id: 'taskTitleInput', action: () => this.handleCreateTask() },
            { id: 'timeEntryStartDateInput', action: () => this.handleCreateTimeEntry() },
            { id: 'timeEntryStartTimeInput', action: () => this.handleCreateTimeEntry() },
            { id: 'timeEntryEndDateInput', action: () => this.handleCreateTimeEntry() },
            { id: 'timeEntryEndTimeInput', action: () => this.handleCreateTimeEntry() },
            { id: 'auditEfficiencyRating', action: () => this.handleCreateAudit() },
            { id: 'imagePhotoType', action: () => this.handleUploadImage() },
        ];

        enterTargets.forEach(({ id, action }) => {
            const input = document.getElementById(id);
            if (!input) return;
            input.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    action();
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
        this.showPage('jobs');
        this.showTaskSubpage('time');
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
        const startDate = document.getElementById('timeEntryStartDateInput').value;
        const startTime = document.getElementById('timeEntryStartTimeInput').value;
        const endDate = document.getElementById('timeEntryEndDateInput').value;
        const endTime = document.getElementById('timeEntryEndTimeInput').value;
        const note = document.getElementById('timeEntryNoteInput').value.trim();
        UIManager.setError('timeEntryError', '');

        if (!this.currentTask) {
            UIManager.setError('timeEntryError', 'Select a task before logging time');
            return;
        }

        if (!startDate || !startTime) {
            UIManager.setError('timeEntryError', 'Start date and start time are required');
            return;
        }

        if ((endDate && !endTime) || (!endDate && endTime)) {
            UIManager.setError('timeEntryError', 'Both end date and end time are required or leave both blank');
            return;
        }

        const start = `${startDate}T${startTime}:00`;
        const end = endDate && endTime ? `${endDate}T${endTime}:00` : null;

        try {
            await APIClient.createTimeEntry(this.currentTask.id, start, end, note);
            UIManager.showToast('Time entry saved');
            document.getElementById('timeEntryStartDateInput').value = '';
            document.getElementById('timeEntryStartTimeInput').value = '';
            document.getElementById('timeEntryEndDateInput').value = '';
            document.getElementById('timeEntryEndTimeInput').value = '';
            document.getElementById('timeEntryNoteInput').value = '';
            await this.loadTimeEntries();
        } catch (error) {
            UIManager.setError('timeEntryError', error.message);
        }
    }

    async handleStartTimer() {
        UIManager.setError('timeEntryError', '');

        if (!this.currentTask) {
            UIManager.setError('timeEntryError', 'Select a task before starting a timer');
            return;
        }

        const activeEntry = this.timeEntries.find((entry) => !entry.endTime);
        if (activeEntry) {
            UIManager.setError('timeEntryError', 'Stop the active timer before starting another one');
            return;
        }

        const note = document.getElementById('timeEntryNoteInput').value.trim();
        const start = new Date().toISOString();

        try {
            await APIClient.createTimeEntry(this.currentTask.id, start, null, note);
            UIManager.showToast('Timer started');
            document.getElementById('timeEntryNoteInput').value = '';
            document.getElementById('timeEntryStartInput').value = '';
            document.getElementById('timeEntryEndInput').value = '';
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
            const isSelected = this.currentJob?.id === job.id;
            const item = document.createElement('button');
            item.type = 'button';
            item.className = `job-item ${isSelected ? 'selected' : ''}`;
            item.innerHTML = `
                <div>
                    <div class="job-title">${UIManager.escapeHtml(job.title)}</div>
                    ${isSelected && job.description ? `<p class="job-description">${UIManager.escapeHtml(job.description)}</p>` : ''}
                </div>
                <div class="job-meta">
                    ${isSelected ? '<span class="job-badge">Selected</span>' : ''}
                </div>
            `;
            item.addEventListener('click', () => this.selectJob(job));
            list.appendChild(item);
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
            const isSelected = this.currentTask?.id === task.id;
            const item = document.createElement('button');
            item.type = 'button';
            item.className = `task-item ${isSelected ? 'selected' : ''}`;
            item.innerHTML = `
                <div>
                    <div class="task-title">${UIManager.escapeHtml(task.title)}</div>
                    <p class="task-description">${UIManager.escapeHtml(task.description || 'No description added')}</p>
                    <div class="task-meta">ID: ${UIManager.escapeHtml(task.id.substring(0, 8))}</div>
                </div>
                <div class="task-actions">
                    ${isSelected ? '<span class="task-badge">Selected</span>' : '<span class="task-status">Select</span>'}
                </div>
            `;
            item.addEventListener('click', () => this.selectTask(task));
            list.appendChild(item);
        });
    }

    renderTimeEntries() {
        const list = document.getElementById('timeEntriesList');
        list.innerHTML = '';

        const summary = document.getElementById('taskTimeSummary');
        summary.classList.add('hidden');
        summary.innerHTML = '';

        if (!this.currentTask) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'Select a task to see time entries.';
            list.appendChild(empty);
            return;
        }

        const totalMinutes = this.timeEntries.reduce((sum, entry) => sum + entry.durationMinutes, 0);
        const activeEntry = this.timeEntries.find((entry) => !entry.endTime);

        summary.classList.remove('hidden');
        summary.innerHTML = `
            <h3>${UIManager.escapeHtml(this.currentTask.title)}</h3>
            <p>${UIManager.escapeHtml(this.currentTask.description || 'No description added')}</p>
            <p><strong>Total tracked time:</strong> ${this.formatDuration(totalMinutes)}</p>
            <p><strong>Task ID:</strong> ${UIManager.escapeHtml(this.currentTask.id.substring(0, 8))}</p>
            ${activeEntry ? `<p><strong>Active session started:</strong> ${UIManager.escapeHtml(new Date(activeEntry.startTime).toLocaleString())}</p>` : ''}
        `;

        if (this.timeEntries.length === 0) {
            const empty = document.createElement('div');
            empty.className = 'empty-state';
            empty.textContent = 'No entries logged yet.';
            list.appendChild(empty);
            return;
        }

        const table = document.createElement('table');
        table.className = 'time-table';
        table.innerHTML = `
            <thead>
                <tr>
                    <th>Start</th>
                    <th>End</th>
                    <th>Duration</th>
                    <th>User</th>
                    <th>Note</th>
                </tr>
            </thead>
            <tbody></tbody>
        `;
        const tbody = table.querySelector('tbody');

        const userLabel = (entry) => {
            if (entry.userFirstName) {
                return this.currentUser?.id === entry.userId ? 'You' : entry.userFirstName;
            }
            if (entry.userId) {
                return `${entry.userId.substring(0, 8)}`;
            }
            return 'Unknown';
        };

        this.timeEntries.forEach((entry) => {
            const row = document.createElement('tr');
            row.innerHTML = `
                <td>${UIManager.escapeHtml(new Date(entry.startTime).toLocaleString())}</td>
                <td>${entry.endTime ? UIManager.escapeHtml(new Date(entry.endTime).toLocaleString()) : 'In progress'}</td>
                <td>${UIManager.escapeHtml(this.formatDuration(entry.durationMinutes))}</td>
                <td>${UIManager.escapeHtml(userLabel(entry))}</td>
                <td>${UIManager.escapeHtml(entry.note || '')}</td>
            `;
            tbody.appendChild(row);
        });

        list.appendChild(table);
    }

    formatDuration(totalMinutes) {
        const hours = Math.floor(totalMinutes / 60);
        const minutes = totalMinutes % 60;
        return `${hours}h ${minutes}m`;
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
            const savedBy = audit.userId && this.currentUser?.id === audit.userId
                ? 'You'
                : audit.userFirstName || (audit.userId ? audit.userId.substring(0, 8) : 'Unknown');
            const card = document.createElement('div');
            card.className = 'list-card';
            card.innerHTML = `
                <h3>Audit at ${UIManager.escapeHtml(new Date(audit.createdAt).toLocaleString())}</h3>
                <p><strong>Saved by:</strong> ${UIManager.escapeHtml(savedBy)}</p>
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
            card.className = 'list-card image-card';
            const createdAt = new Date(image.createdAt).toLocaleString();
            const previewImage = image.mimeType && image.mimeType.startsWith('image/')
                ? `<img class="image-preview" src="${UIManager.escapeHtml(`${window.location.origin}/api/images/${encodeURIComponent(image.id)}`)}" alt="${UIManager.escapeHtml(image.originalName || 'Uploaded image')}" />`
                : '';

            card.innerHTML = `
                <div class="image-card-main">
                    ${previewImage}
                    <div>
                        <h3>${UIManager.escapeHtml(image.photoType || 'Photo')}</h3>
                        <p>${UIManager.escapeHtml(image.originalName || image.fileName || 'Uploaded image')}</p>
                        <p>${UIManager.escapeHtml(createdAt)}</p>
                        <p>${UIManager.escapeHtml(image.mimeType || 'Unknown type')}</p>
                    </div>
                </div>
                <div class="image-card-actions">
                    <button class="btn btn-small" data-image-id="${image.id}">View</button>
                    <button class="btn btn-small" data-download-id="${image.id}">Download</button>
                </div>
            `;
            card.querySelector('[data-image-id]').addEventListener('click', () => this.viewImage(image.id));
            card.querySelector('[data-download-id]').addEventListener('click', () => this.downloadImage(image.id, image.originalName || 'photo'));
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

    viewImage(imageId) {
        const url = `${window.location.origin}/api/images/${encodeURIComponent(imageId)}`;
        window.open(url, '_blank');
    }

    async selectJob(job) {
        this.currentJob = job;
        document.getElementById('selectedJobTitle').textContent = `Selected Job: ${job.title}`;
        await this.loadTasks();
        await this.loadJobs();
        this.clearTaskSelection();
        this.showPage('tasks');
    }

    async selectTask(task) {
        this.currentTask = task;
        document.getElementById('selectedTaskTitle').textContent = `Selected Task: ${task.title}`;
        await Promise.all([this.loadTimeEntries(), this.loadAudits(), this.loadImages()]);
        this.showPage('taskDetail');
        this.showTaskSubpage('time');
        this.setTimeEntryDefaultDates();
    }

    clearJobSelection() {
        this.currentJob = null;
        document.getElementById('selectedJobTitle').textContent = 'Select a job to manage tasks.';
        this.tasks = [];
        this.renderTasksList();
        this.currentTask = null;
        document.getElementById('selectedTaskTitle').textContent = 'Select a task to log time.';
        this.timeEntries = [];
        this.audits = [];
        this.images = [];
        this.renderTimeEntries();
        this.renderAudits();
        this.renderImages();
        this.showPage('jobs');
        this.showTaskSubpage('time');
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
        this.showPage('tasks');
        this.showTaskSubpage('time');
        this.setTimeEntryDefaultDates();
    }

    showPage(page) {
        this.currentPage = page;

        document.getElementById('jobsPage').classList.toggle('hidden', page !== 'jobs');
        document.getElementById('tasksPage').classList.toggle('hidden', page !== 'tasks');
        document.getElementById('taskDetailPage').classList.toggle('hidden', page !== 'taskDetail');
    }

    showTaskSubpage(subpage) {
        this.currentTaskPage = subpage;
        document.getElementById('timeSubpage').classList.toggle('hidden', subpage !== 'time');
        document.getElementById('auditSubpage').classList.toggle('hidden', subpage !== 'audit');
        document.getElementById('photoSubpage').classList.toggle('hidden', subpage !== 'photo');

        if (subpage === 'time') {
            this.setTimeEntryDefaultDates();
        }

        ['showTimePageBtn', 'showAuditPageBtn', 'showPhotoPageBtn'].forEach((buttonId) => {
            const button = document.getElementById(buttonId);
            if (!button) return;
            button.classList.toggle('active', buttonId === `show${subpage.charAt(0).toUpperCase() + subpage.slice(1)}PageBtn`);
        });
    }

    setTimeEntryDefaultDates() {
        const now = new Date();
        const localDate = now.toISOString().split('T')[0];
        const localTime = now.toTimeString().slice(0, 5);
        const startDateInput = document.getElementById('timeEntryStartDateInput');
        const endDateInput = document.getElementById('timeEntryEndDateInput');
        const startTimeInput = document.getElementById('timeEntryStartTimeInput');
        const endTimeInput = document.getElementById('timeEntryEndTimeInput');

        if (startDateInput && !startDateInput.value) {
            startDateInput.value = localDate;
        }
        if (endDateInput && !endDateInput.value) {
            endDateInput.value = localDate;
        }
        if (startTimeInput && !startTimeInput.value) {
            startTimeInput.value = localTime;
        }
        if (endTimeInput && !endTimeInput.value) {
            endTimeInput.value = localTime;
        }
    }

    toggleSection(sectionId, isVisible) {
        const section = document.getElementById(sectionId);
        if (!section) return;
        section.classList.toggle('hidden', !isVisible);
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
