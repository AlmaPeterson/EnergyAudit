// ============================================================================
// UI Manager
// ============================================================================

class UIManager {
    static showAuthSection() {
        document.getElementById('authSection').classList.remove('hidden');
        document.getElementById('appSection').classList.add('hidden');
    }

    static showAppSection() {
        document.getElementById('authSection').classList.add('hidden');
        document.getElementById('appSection').classList.remove('hidden');
    }

    static setLoading(elementId, isLoading) {
        const element = document.getElementById(elementId);
        if (!element) return;
        
        if (isLoading) {
            element.classList.remove('hidden');
        } else {
            element.classList.add('hidden');
        }
    }

    static setError(elementId, message) {
        const element = document.getElementById(elementId);
        if (!element) return;
        
        if (message) {
            element.textContent = message;
            element.classList.remove('hidden');
        } else {
            element.textContent = '';
            element.classList.add('hidden');
        }
    }

    static showToast(message, type = 'success') {
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        toast.textContent = message;
        document.body.appendChild(toast);

        setTimeout(() => {
            toast.classList.add('show');
        }, 10);

        setTimeout(() => {
            toast.classList.remove('show');
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }

    static escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
}
