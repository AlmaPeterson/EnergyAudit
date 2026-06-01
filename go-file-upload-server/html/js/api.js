// ============================================================================
// API Client
// ============================================================================

class APIClient {
    static async extractError(response) {
        try {
            const error = await response.json();
            return error.message || error.error || JSON.stringify(error);
        } catch (jsonError) {
            try {
                return await response.text();
            } catch (textError) {
                return `HTTP ${response.status}`;
            }
        }
    }

    static async request(endpoint, options = {}) {
        const headers = {
            'Content-Type': 'application/json',
            ...options.headers,
            ...AuthManager.getAuthHeader(),
        };

        const config = {
            ...options,
            headers,
        };

        try {
            const response = await fetch(`${API_BASE}${endpoint}`, config);

            if (response.status === 401) {
                AuthManager.clearToken();
                window.location.reload();
                return;
            }

            if (!response.ok) {
                const errorMessage = await this.extractError(response);
                throw new Error(errorMessage || `HTTP ${response.status}`);
            }

            if (response.status === 204) {
                return null;
            }

            const contentType = response.headers.get('Content-Type') || '';
            if (contentType.includes('application/json')) {
                return await response.json();
            }

            return await response.text();
        } catch (error) {
            console.error('API Error:', error);
            throw error;
        }
    }

    // Authentication
    static async signup(firstName, lastName, email, password) {
        return this.request('/signup', {
            method: 'POST',
            body: JSON.stringify({ firstName, lastName, email, password }),
        });
    }

    static async login(email, password) {
        return this.request('/login', {
            method: 'POST',
            body: JSON.stringify({ email, password }),
        });
    }

    // Jobs
    static async getJobs() {
        return this.request('/jobs');
    }

    static async createJob(title, description) {
        return this.request('/jobs', {
            method: 'POST',
            body: JSON.stringify({ title, description }),
        });
    }

    // Tasks
    static async listTasks(jobId) {
        return this.request(`/tasks?jobId=${encodeURIComponent(jobId)}`);
    }

    static async getTask(id) {
        return this.request(`/tasks/${encodeURIComponent(id)}`);
    }

    static async createTask(jobId, title, description) {
        return this.request('/tasks', {
            method: 'POST',
            body: JSON.stringify({ jobId, title, description }),
        });
    }

    static async updateTask(taskId, data) {
        return this.request(`/tasks/${encodeURIComponent(taskId)}`, {
            method: 'PUT',
            body: JSON.stringify(data),
        });
    }

    // Time entries
    static async listTimeEntries(taskId) {
        return this.request(`/time-entries?taskId=${encodeURIComponent(taskId)}`);
    }

    static async createTimeEntry(taskId, startTime, endTime, note) {
        return this.request('/time-entries', {
            method: 'POST',
            body: JSON.stringify({ taskId, startTime, endTime, note }),
        });
    }

    static async stopTimeEntry(entryId) {
        return this.request(`/time-entries/${encodeURIComponent(entryId)}/stop`, {
            method: 'PUT',
        });
    }

    // Energy audits
    static async listEnergyAudits(taskId) {
        return this.request(`/audits?taskId=${encodeURIComponent(taskId)}`);
    }

    static async createEnergyAudit(jobId, taskId, easy, hard, fun, notFun, efficiencyRating, notes) {
        return this.request('/audits', {
            method: 'POST',
            body: JSON.stringify({ jobId, taskId, easy, hard, fun, notFun, efficiencyRating, notes }),
        });
    }

    // Images
    static async uploadImage(file, taskId, jobId, photoType) {
        const formData = new FormData();
        formData.append('image', file);
        formData.append('taskId', taskId);
        if (jobId) {
            formData.append('jobId', jobId);
        }
        formData.append('photoType', photoType);

        const response = await fetch(`${API_BASE}/images`, {
            method: 'POST',
            body: formData,
            headers: AuthManager.getAuthHeader(),
        });

        if (response.status === 401) {
            AuthManager.clearToken();
            window.location.reload();
            return;
        }

        if (!response.ok) {
            const errorMessage = await this.extractError(response);
            throw new Error(errorMessage || `HTTP ${response.status}`);
        }

        return response.json();
    }

    static async listImages(taskId) {
        return this.request(`/images?taskId=${encodeURIComponent(taskId)}`);
    }

    static async downloadImage(id) {
        const response = await fetch(`${API_BASE.replace('/api', '')}/images/${encodeURIComponent(id)}`, {
            method: 'GET',
            headers: AuthManager.getAuthHeader(),
        });

        if (!response.ok) {
            const errorMessage = await this.extractError(response);
            throw new Error(errorMessage || `HTTP ${response.status}`);
        }

        return await response.blob();
    }
}
