document.getElementById('verify-token-btn').addEventListener('click', async () => {
    const token = document.getElementById('reset-token').value;

    const res = await fetch('http://localhost:8080/verify-reset-token', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token })
    });

    if (res.ok) {
        document.getElementById('step-1').style.display = 'none';
        document.getElementById('step-2').style.display = 'block';
        localStorage.setItem('reset_token', token);
    } else {
        alert('Неверный или просроченный токен');
    }
});

document.getElementById('reset-password-btn').addEventListener('click', async () => {
    const password = document.getElementById('new-password').value;
    const confirm = document.getElementById('confirm-password').value;
    const token = localStorage.getItem('reset_token');

    if (password !== confirm) {
        alert('Пароли не совпадают');
        return;
    }

    const res = await fetch('http://localhost:8080/reset-password', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ token, new_password: password })
    });

    if (res.ok) {
        alert('Пароль успешно изменён. Теперь вы можете войти.');
        window.location.href = 'auth.html';
    } else {
        alert('Ошибка при сбросе пароля');
    }
});