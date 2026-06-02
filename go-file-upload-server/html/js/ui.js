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

    static initTheme() {
        try {
            const saved = localStorage.getItem('theme');
            const prefersLight = window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches;
            const theme = saved || (prefersLight ? 'light' : 'dark');
            this.applyTheme(theme);
            // Use delegated listeners so the handler works on mobile and if the button is re-rendered
            const handler = (e) => {
                const btn = e.target.closest && e.target.closest('#themeToggle');
                if (btn) {
                    this.toggleTheme();
                }
            };
            // regular click
            document.addEventListener('click', handler);
            // touch events on some mobile browsers may behave differently; listen without passive flag and avoid preventDefault
            document.addEventListener('touchstart', handler);

            // set initial label if the element exists
            const toggleBtn = document.getElementById('themeToggle');
            if (toggleBtn) toggleBtn.textContent = theme === 'light' ? 'Light' : 'Dark';
        } catch (e) {
            // ignore
        }
    }

    static applyTheme(theme) {
        if (theme === 'light') {
            document.body.setAttribute('data-theme', 'light');
        } else {
            document.body.removeAttribute('data-theme');
        }
        try { localStorage.setItem('theme', theme); } catch (e) {}
        const toggle = document.getElementById('themeToggle');
        if (toggle) toggle.textContent = theme === 'light' ? 'Light' : 'Dark';
    }

    static toggleTheme() {
        const current = document.body.getAttribute('data-theme') === 'light' ? 'light' : 'dark';
        const next = current === 'light' ? 'dark' : 'light';
        this.applyTheme(next);
    }
}
