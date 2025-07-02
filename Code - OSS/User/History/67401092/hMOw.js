document.addEventListener('DOMContentLoaded', () => {
    const token = localStorage.getItem('access_token');
    if (!token) {
        window.location.href = 'auth.html';
        return;
    }

    const userEmail = localStorage.getItem('user_email');
    if (userEmail) {
        document.getElementById('user-email').textContent = userEmail;
    }

    const navLinks = document.querySelectorAll('.nav-link');
    const pages = document.querySelectorAll('.page');

    navLinks.forEach(link => {
        link.addEventListener('click', () => {
            const pageId = link.getAttribute('data-page');

            navLinks.forEach(l => l.classList.remove('active'));
            link.classList.add('active');

            pages.forEach(page => page.classList.remove('active'));
            document.getElementById(`${pageId}-page`).classList.add('active');

            switch(pageId) {
                case 'history':
                    loadNotificationHistory();
                    break;
                case 'stats':
                    loadStatistics();
                    break;
            }
        });
    });

    const channelBtns = document.querySelectorAll('.channel-btn');
    channelBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            channelBtns.forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            updateRecipientHint(btn.getAttribute('data-channel'));
        });
    });

    document.getElementById('send-btn').addEventListener('click', sendNotification);

    document.getElementById('logout-btn').addEventListener('click', logout);

    updateRecipientHint('email');

    loadNotificationHistory();
});

function updateRecipientHint(channel) {
    const hint = document.getElementById('recipient-hint');

    switch(channel) {
        case 'email':
            hint.innerHTML = 'Для email: user@example.com<br>' +
                             'Пример: john.doe@company.com';
            break;
        case 'telegram':
            hint.innerHTML = 'Для Telegram: ID чата (числовой идентификатор)<br>' +
                             'Пример: 1234567890';
            break;
        case 'whatsapp':
            hint.innerHTML = 'Для WhatsApp: номер телефона в международном формате<br>' +
                             'Пример: 79150000000 (без пробелов и спецсимволов)';
            break;
    }
}

async function sendNotification() {
    const activeChannel = document.querySelector('.channel-btn.active').getAttribute('data-channel');
    const recipient = document.getElementById('recipient').value;
    const subject = document.getElementById('subject').value;
    const message = document.getElementById('message').value;

    if (!recipient || !message) {
        showNotification('Заполните обязательные поля', 'warning');
        return;
    }

    try {
        const response = await fetch('http://localhost:8081/notify', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${localStorage.getItem('access_token')}`
            },
            body: JSON.stringify({
                channel: activeChannel,
                recipient: recipient,
                subject: subject,
                body: message
            })
        });

        if (response.ok) {
            showNotification('Уведомление поставлено в очередь на отправку', 'success');
            document.getElementById('recipient').value = '';
            document.getElementById('subject').value = '';
            document.getElementById('message').value = '';

            loadNotificationHistory();
        } else {
            const data = await response.json();
            showNotification(data.error || 'Ошибка при отправке уведомления', 'error');
        }
    } catch (error) {
        showNotification('Ошибка соединения с сервером', 'error');
        console.error('Send notification error:', error);
    }
}


async function loadNotificationHistory() {
    const limit = document.getElementById('history-limit').value;
    const channel = document.getElementById('history-channel').value;

    let url = `http://localhost:8081/history?limit=${limit}`;
    if (channel && channel !== 'all') {
        url += `&channel=${channel}`;
    }

    try {
        const response = await fetch(url, {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('access_token')}`
            }
        });

        if (response.ok) {
            const history = await response.json();
            renderHistoryTable(history);
        } else {
            showNotification('Ошибка загрузки истории', 'error');
        }
    } catch (error) {
        showNotification('Ошибка соединения с сервером', 'error');
        console.error('Load history error:', error);
    }
}

function renderHistoryTable(history) {
    const tableBody = document.querySelector('#history-table tbody');
    tableBody.innerHTML = '';

    history.forEach(notification => {
        const row = document.createElement('tr');

        const date = new Date(notification.created_at);
        const formattedDate = `${date.toLocaleDateString()} ${date.toLocaleTimeString()}`;

        let statusIcon = '🟡';
        if (notification.status === 'sent') statusIcon = '🟢';
        if (notification.status === 'failed') statusIcon = '🔴';

        row.innerHTML = `
            <td>${formattedDate}</td>
            <td>${notification.channel}</td>
            <td>${notification.recipient}</td>
            <td>${notification.subject || '-'}</td>
            <td>${statusIcon} ${notification.status}</td>
        `;

        tableBody.appendChild(row);
    });
}

async function loadStatistics() {
    try {
        const response = await fetch('http://localhost:8081/stats', {
            method: 'GET',
            headers: {
                'Authorization': `Bearer ${localStorage.getItem('access_token')}`
            }
        });

        if (response.ok) {
            const stats = await response.json();
            renderStatistics(stats);
        } else {
            showNotification('Ошибка загрузки статистики', 'error');
        }
    } catch (error) {
        showNotification('Ошибка соединения с сервером', 'error');
        console.error('Load statistics error:', error);
    }
}

function renderStatistics(stats) {
    document.getElementById('total-notifications').textContent = stats.total || 0;
    document.getElementById('email-count').textContent = stats.email_count || 0;
    document.getElementById('telegram-count').textContent = stats.telegram_count || 0;
    document.getElementById('whatsapp-count').textContent = stats.whatsapp_count || 0;

    const successRate = stats.sent && stats.total ?
        Math.round((stats.sent / stats.total) * 100) : 0;
    document.getElementById('success-rate').textContent = `${successRate}%`;

    renderChannelChart(stats);

    renderStatusChart(stats);
}

function renderChannelChart(stats) {
    const ctx = document.getElementById('channel-chart').getContext('2d');

    }

    window.channelChart = new Chart(ctx, {
        type: 'pie',
        data: {
            labels: ['Email', 'Telegram', 'WhatsApp'],
            datasets: [{
                data: [
                    stats.email_count || 0,
                    stats.telegram_count || 0,
                    stats.whatsapp_count || 0
                ],
                backgroundColor: [
                    '#4361ee',
                    '#3f37c9',
                    '#4cc9f0'
                ],
                borderWidth: 1
            }]
        },
        options: {
            responsive: true,
            plugins: {
                legend: {
                    position: 'right'
                }
            }
        }
    });
}

function renderStatusChart(stats) {
    const ctx = document.getElementById('status-chart').getContext('2d');

    if (window.statusChart) {
        window.statusChart.destroy();
    }

    window.statusChart = new Chart(ctx, {
        type: 'doughnut',
        data: {
            labels: ['Успешно', 'В очереди', 'Ошибка'],
            datasets: [{
                data: [
                    stats.sent || 0,
                    stats.queued || 0,
                    stats.failed || 0
                ],
                backgroundColor: [
                    '#4cc9f0', // sent
                    '#f8961e', // queued
                    '#f72585'  // failed
                ],
                borderWidth: 1
            }]
        },
        options: {
            responsive: true,
            plugins: {
                legend: {
                    position: 'right'
                }
            }
        }
    });
}

function logout() {
    localStorage.removeItem('access_token');
    localStorage.removeItem('user_email');
    window.location.href = 'index.html';
}

function showNotification(message, type) {
    const notification = document.getElementById('notification');
    notification.textContent = message;
    notification.className = `notification ${type} show`;

    setTimeout(() => {
        notification.classList.remove('show');
    }, 5000);
}