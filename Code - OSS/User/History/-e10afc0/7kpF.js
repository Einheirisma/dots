document.addEventListener('DOMContentLoaded', () => {
    const tabs = document.querySelectorAll('.tab');
    const forms = document.querySelectorAll('.auth-form');

    tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            tabs.forEach(t => t.classList.remove('active'));
            forms.forEach(f => f.classList.remove('active'));
            tab.classList.add('active');
            const tabId = tab.getAttribute('data-tab');
            document.getElementById(`${tabId}-form`).classList.add('active');
        });
    });

    document.querySelector('.auth-link').addEventListener('click', function () {
        const showForm = this.getAttribute('data-show');
        tabs.forEach(t => t.classList.remove('active'));
        forms.forEach(f => f.classList.remove('active'));
        document.querySelector(`[data-tab="${showForm}"]`).classList.add('active');
        document.getElementById(`${showForm}-form`).classList.add('active');
    });

    document.getElementById('register-btn').addEventListener('click', register);
    document.getElementById('login-btn').addEventListener('click', login);
    document.getElementById('reset-btn').addEventListener('click', resetPassword);
    document.getElementById('verify-btn').addEventListener('click', verifyEmail);
    document.getElementById('resend-btn').addEventListener('click', resendVerification);

    document.getElementById('verify-token-btn').addEventListener('click', async () => {
        const token = document.getElementById('reset-token').value;
        try {
            const response = await fetch('http://localhost:8080/verify-reset-token', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ token })
            });

            if (response.ok) {
                localStorage.setItem('reset_token', token);
                showForm('new-password-form');
            } else {
                const data = await response.json();
                showNotification(data.error || 'Неверный или просроченный код', 'error');
            }
        } catch (error) {
            showNotification('Ошибка соединения с сервером', 'error');
            console.error('Verify token error:', error);
        }
    });

    document.getElementById('set-password-btn').addEventListener('click', async () => {
        const password = document.getElementById('new-password').value;
        const confirmPassword = document.getElementById('confirm-password').value;
        const token = localStorage.getItem('reset_token');

        if (password !== confirmPassword) {
            showNotification('Пароли не совпадают', 'error');
            return;
        }

        try {
            const response = await fetch('http://localhost:8080/reset-password', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ token, password, confirm_password: confirmPassword })
            });

            if (response.ok) {
                showNotification('Пароль успешно изменен', 'success');
                localStorage.removeItem('reset_token');
                setTimeout(() => {
                    showForm('login-form');
                    document.querySelector('[data-tab="login"]').classList.add('active');
                }, 2000);
            } else {
                const data = await response.json();
                showNotification(data.error || 'Ошибка при смене пароля', 'error');
            }
        } catch (error) {
            showNotification('Ошибка соединения с сервером', 'error');
            console.error('Set password error:', error);
        }
    });
});

async function register() {
    const email = document.getElementById('register-email').value;
    const password = document.getElementById('register-password').value;
    const confirm = document.getElementById('register-confirm').value;

    if (password !== confirm) {
        showNotification('Пароли не совпадают', 'error');
        return;
    }

    try {
        const response = await fetch('http://localhost:8080/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        });

        const data = await response.json();

        if (response.ok) {
            document.querySelectorAll('.auth-form').forEach(f => f.classList.remove('active'));
            document.getElementById('verify-form').classList.add('active');
            document.getElementById('verify-form').dataset.email = email;
            showNotification('Регистрация успешна! Проверьте вашу почту для подтверждения.', 'success');
        } else {
            showNotification(data.error || 'Ошибка регистрации', 'error');
        }
    } catch (error) {
        showNotification('Ошибка соединения с сервером', 'error');
        console.error('Register error:', error);
    }
}

async function login() {
    const email = document.getElementById('login-email').value;
    const password = document.getElementById('login-password').value;

    try {
        const response = await fetch('http://localhost:8080/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email, password })
        });

        const data = await response.json();

        if (response.ok) {
            localStorage.setItem('access_token', data.access_token);
            localStorage.setItem('user_email', email);
            window.location.href = 'dashboard.html';
        } else {
            showNotification(data.error || 'Ошибка входа', 'error');
        }
    } catch (error) {
        showNotification('Ошибка соединения с сервером', 'error');
        console.error('Login error:', error);
    }
}

async function resetPassword() {
    const email = document.getElementById('reset-email').value;

    try {
        const response = await fetch('http://localhost:8080/forgot-password', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email })
        });

        if (response.ok) {
            showNotification('Инструкции по сбросу пароля отправлены на вашу почту', 'success');
            showForm('reset-token-form');
        } else {
            const data = await response.json();
            showNotification(data.error || 'Ошибка при запросе сброса пароля', 'error');
        }
    } catch (error) {
        showNotification('Ошибка соединения с сервером', 'error');
        console.error('Reset password error:', error);
    }
}

async function verifyEmail() {
    const token = document.getElementById('verify-token').value;
    const email = document.getElementById('verify-form').dataset.email;

    try {
        const response = await fetch(`http://localhost:8080/verify-email?token=${token}`, {
            method: 'GET'
        });

        if (response.ok) {
            showNotification('Email успешно подтвержден! Теперь вы можете войти.', 'success');
            setTimeout(() => {
                document.querySelectorAll('.auth-form').forEach(f => f.classList.remove('active'));
                document.getElementById('login-form').classList.add('active');
                document.querySelector('[data-tab="login"]').classList.add('active');
            }, 2000);
        } else {
            const data = await response.json();
            showNotification(data.error || 'Неверный токен подтверждения', 'error');
        }
    } catch (error) {
        showNotification('Ошибка соединения с сервером', 'error');
        console.error('Verify email error:', error);
    }
}

async function resendVerification() {
    const email = document.getElementById('verify-form').dataset.email;

    try {
        const response = await fetch('http://localhost:8080/resend-verification', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ email })
        });

        if (response.ok) {
            showNotification('Новый код подтверждения отправлен на вашу почту', 'success');
        } else {
            const data = await response.json();
            showNotification(data.error || 'Ошибка при отправке кода', 'error');
        }
    } catch (error) {
        showNotification('Ошибка соединения с сервером', 'error');
        console.error('Resend verification error:', error);
    }
}

function showNotification(message, type) {
    const notification = document.getElementById('notification');
    notification.textContent = message;
    notification.className = `notification ${type} show`;
    setTimeout(() => {
        notification.classList.remove('show');
    }, 5000);
}

function showForm(formId) {
    document.querySelectorAll('.auth-form').forEach(f => f.classList.remove('active'));
    document.getElementById(formId).classList.add('active');
}