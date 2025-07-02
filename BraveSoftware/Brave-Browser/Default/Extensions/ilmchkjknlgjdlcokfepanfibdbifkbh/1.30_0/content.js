// content.js

let startTime = null;
let timerInterval = null;
let waitingForResponse = false;
let currentUrl = window.location.href;
let timeoutStartTime = null;
let timeoutInterval = null;
let timeoutDuration = 0;
let originalTitle = document.title;
let blinkInterval;
let currentTabId;
let errorCheckInterval = null;
let regenerateButtonCheckInterval = null;
let isCheckingForRegenerateButton = false;
let regenerateTimeoutActive = false;


// Get the tab ID when the script loads (with error handling)
chrome.runtime.sendMessage({ action: "getCurrentTabId" }, (response) => {
    if (chrome.runtime.lastError) {
        console.error("Runtime error:", chrome.runtime.lastError.message);
        return;
    }
    if (response?.tabId) {
        currentTabId = response.tabId;
        console.log("DeepSeek Server Busy: Initialized Tab ID:", currentTabId);
    } else {
        console.error("DeepSeek Server Busy: Failed to get Tab ID. Response:", response);
    }
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === "resetNotificationState") {
        console.log("RESETTING STATE - Tab ID:", currentTabId);
        clearInterval(blinkInterval);
        clearInterval(errorCheckInterval);
        document.title = originalTitle;
        changeFavicon('default');
        sendResponse({ status: "success" });
        return true;
    }
});

// Функция для мигания
function startBlinking(wasHiddenWhenCompleted) {
    if (blinkInterval) clearInterval(blinkInterval);
    let isBlinking = false;
    const alertIcon = chrome.runtime.getURL('red.svg');

    // Use the captured visibility state at completion time
    let shouldBlink = wasHiddenWhenCompleted;
    
    blinkInterval = setInterval(() => {
        // Only blink if tab was hidden when response arrived
        if (!shouldBlink) return;
        
        isBlinking = !isBlinking;
        document.title = isBlinking 
            ? "🚨 Response Ready! 🚨" 
            : originalTitle;
        changeFavicon(isBlinking ? 'red' : 'default');
    }, 1000);

    // Update logic for visibility changes
    const handleVisibilityChange = () => {
        if (document.hidden) return;
        // If user returns to tab, stop blinking
        clearInterval(blinkInterval);
        document.title = originalTitle;
        changeFavicon('default');
        document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);
}

// Словарь фраз
//const phrases = [
//    "The server is busy. Please try again later.",
//    "服务器繁忙，请稍后再试"
//];

function startTimer() {
    startTime = Date.now();
    timerInterval = setInterval(updateTimer, 500);
}

function stopTimer() {
    if (timerInterval) {
        clearInterval(timerInterval);
        timerInterval = null;
    }
    startTime = null;
    document.title = 'DeepSeek w/ "DeepSeek Server Busy" extension';
}

function updateTimer() {
    if (!startTime) return;
    let elapsed = (Date.now() - startTime) / 1000;
    elapsed = Math.round(elapsed * 10) / 10; // Округление до 0.1 сек
    // Полностью заменяем заголовок вместо добавления
    document.title = `${elapsed}s | ${originalTitle}`;
}

function startTimeoutTimer(delay) {
    timeoutDuration = delay;
    timeoutStartTime = Date.now();
    timeoutInterval = setInterval(updateTimeoutTimer, 500);
    regenerateTimeoutActive = false; // Reset this flag
    changeFavicon('gray');
    console.log(`Waiting for timeout: ${delay} seconds. You can change the timeout by clicking the plugin icon.`);
}

function stopTimeoutTimer() {
    if (timeoutInterval) {
        clearInterval(timeoutInterval);
        timeoutInterval = null;
    }
    if (errorCheckInterval) {
        clearInterval(errorCheckInterval);
        errorCheckInterval = null;
    }
    timeoutStartTime = null;
    timeoutDuration = 0;
    regenerateTimeoutActive = false;
    document.title = originalTitle;
    document.querySelectorAll('.timeout-display').forEach(el => el.remove());
    changeFavicon('default');
}


function updateTimeoutTimer() {
    if (!timeoutStartTime || !timeoutDuration) return;
    let elapsed = (Date.now() - timeoutStartTime) / 1000;
    let remaining = timeoutDuration - elapsed;
    remaining = Math.max(0, Math.round(remaining * 10) / 10); // Округление до 0.1 сек
    
    // Обновляем заголовок
    document.title = `⏳ ${remaining}s | ${originalTitle}`;

    // Создаем или находим элемент для отображения таймера
    let timeoutDisplay = document.querySelector('.timeout-display');
    if (!timeoutDisplay) {
        timeoutDisplay = createTimeoutDisplay();
        if (!timeoutDisplay) return; // Если не нашли место для отображения
    }
    
    timeoutDisplay.textContent = `⏳ ${remaining}s`;
}

function createTimeoutDisplay(targetFlexContainer) {
    // Ищем целевой элемент для размещения таймера
    const targetSpan = document.evaluate(
        "//div[@class='e13328ad']/div[@class='ac2694a7']/span",
        document,
        null,
        XPathResult.FIRST_ORDERED_NODE_TYPE,
        null
    ).singleNodeValue;

    if (!targetSpan) return null;

    // Удаляем предыдущие отображения таймера
    document.querySelectorAll('.timeout-display').forEach(el => el.remove());

    // Создаем элемент для отображения таймера
    const display = document.createElement('span');
    display.className = 'timeout-display';
    display.style.cssText = `
        color: #666;
        font-family: monospace;
        margin-left: 8px;
    `;

    // Вставляем после текста "Server busy"
    targetSpan.parentNode.insertBefore(display, targetSpan.nextSibling);

    return display;
}


let currentFaviconState = null;

function changeFavicon(mode) {
    if (currentFaviconState === mode) return;
    
    document.querySelectorAll('link[rel~="icon"]').forEach(link => link.remove());
    
    const link = document.createElement('link');
    link.type = 'image/svg+xml';
    link.rel = 'shortcut icon';
    link.href = chrome.runtime.getURL(`${mode}.svg?t=${Date.now()}`);
    
    document.head.appendChild(link);
    currentFaviconState = mode;
}

function checkUrlChange() {
    const newUrl = window.location.href;
    if (newUrl !== currentUrl) {
        currentUrl = newUrl;
        changeFavicon('default'); 
    }
}

function showTempBanMessage() {
    const targetDiv = document.evaluate(
        "(//div[contains(@class, 'd7dc56a8')])[last()]//div[contains(@style, 'flex: 1 1 0%')]",
        document,
        null,
        XPathResult.FIRST_ORDERED_NODE_TYPE,
        null
    ).singleNodeValue;

    if (targetDiv && !targetDiv.querySelector('.temp-ban-alert')) {
        document.querySelectorAll('.temp-ban-alert').forEach(el => el.remove());

        const alertDiv = document.createElement('div');
        alertDiv.className = 'temp-ban-alert';
        alertDiv.textContent = '⚠️ Temporary ban or Network error detected. "Retry timeout" could be increased by clicking the plugin icon.';
        alertDiv.style.cssText = 'color: red; font-size: 0.9em; padding: 4px;';
        targetDiv.parentNode.insertBefore(alertDiv, targetDiv.nextSibling);

        const observer = new MutationObserver((mutationsList, observer) => {
            const loadingIcon = document.evaluate("(//div[contains(@class, 'b4e4476b')])[last()]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
            if (loadingIcon) {
                alertDiv.remove();
                observer.disconnect();
            }
        });

        observer.observe(document.body, { childList: true, subtree: true });
    }
}

function clickRegenerateButtonInLastFlexBlock() {
    const buttonToClick = document.evaluate(
        "//div[contains(@class, '_9663006')]//div[contains(@class, 'ds-icon-button') and contains(@class, 'a3b9bd76')]",
        document,
        null,
        XPathResult.FIRST_ORDERED_NODE_TYPE,
        null
    ).singleNodeValue;

    if (buttonToClick) {
        console.log('Regenerate button found, clicking...');
        buttonToClick.click();
        console.log('Regenerate button clicked');
        
        // Проверяем, началась ли загрузка после клика
        setTimeout(() => {
            const loadingIcon = document.evaluate(
                "(//div[contains(@class, 'b4e4476b')])[last()]", 
                document, 
                null, 
                XPathResult.FIRST_ORDERED_NODE_TYPE, 
                null
            ).singleNodeValue;
            
            if (!loadingIcon) {
                console.log('Loading not started after click, showing temp ban message');
                showTempBanMessage();
            }
        }, 3000);
    } else {
        console.log('Regenerate button not found');
    }
}

function handleNotification(elapsed, currentTabId) {
    if (!currentTabId) {
        console.error("Tab ID is missing!");
        return;
    }
    chrome.storage.sync.get(['alertOption'], (data) => {
        const alertOption = data.alertOption || 'inactive';
        const retryCountElement = document.evaluate(
            "(//div[contains(@class, 'd7dc56a8')])[last()]//div[@class='dd7e4fda']",
            document,
            null,
            XPathResult.FIRST_ORDERED_NODE_TYPE,
            null
        ).singleNodeValue;

        let retries = 0;
        if (retryCountElement) {
            const match = retryCountElement.textContent.match(/\d+\s*\/\s*(\d+)/);
            retries = match ? parseInt(match[1]) : 0;
        }

        const shouldNotify = 
            alertOption === 'always' || 
            (alertOption === 'inactive' && document.hidden) ||
            (alertOption === 'retry' && retries > 0);

        if (shouldNotify) {
            originalTitle = document.title;
            const wasHidden = document.hidden;
            startBlinking(wasHidden);

            let alertMessage = `Done\n`;
            if (retries > 3) {
                alertMessage += `'Deepseek Server Busy' extension saved you: ~${retries-1} clicks\n`;
            }
            alertMessage += `Latest attempt took: ${elapsed}s`;

            if (!("Notification" in window)) {
                console.log("DeepSeek Server Busy: Browser doesn't support notifications");
                alert(alertMessage);
            } else {
                if (Notification.permission === "granted") {
                    chrome.runtime.sendMessage({ 
                        action: "showNotification", 
                        message: alertMessage,
                        tabId: currentTabId
                    });
                    console.log(`DeepSeek Server Busy: Sent via notification: ${alertMessage}`);
                } else if (Notification.permission !== "denied") {
                    alertMessage += "\n\nUse the bell icon in the address bar to allow native notifications."
                    alert(alertMessage);
                    console.log(`DeepSeek Server Busy: Sent via alert (no access to notifications was granted): ${alertMessage}`);
                        
                    Notification.requestPermission().then(permission => {
                        if (permission === "granted") {
                            chrome.runtime.sendMessage({ 
                                action: "showNotification", 
                                message: alertMessage,
                                tabId: currentTabId
                            });
                            console.log(`DeepSeek Server Busy: Sent via notification after granted access: ${alertMessage}`);
                        }
                    });
                } else {
                    alert(alertMessage);
                    console.log(`DeepSeek Server Busy: Sent via alert: ${alertMessage}`);
                }
            }
        }
    });
}

function checkForErrorInResponseBlock() {
    // Используем только рабочий селектор
    const errorButton = document.evaluate(
        "//div[contains(@class, '_9663006')]//div[contains(@class, 'ds-icon-button') and contains(@class, 'a3b9bd76')]/div[contains(@class, 'ds-icon')]",
        document,
        null,
        XPathResult.FIRST_ORDERED_NODE_TYPE,
        null
    ).singleNodeValue;

    if (errorButton) {
        const elapsed = startTime ? ((Date.now() - startTime) / 1000).toFixed(1) : '0';
        console.log(`DeepSeek Server Busy: Error button detected after ${elapsed}s`);
        
        changeFavicon('red');
        stopTimer();
        
        chrome.storage.sync.get(['minDelay', 'maxDelay'], function (data) {
            const minDelay = data.minDelay || 30;
            const maxDelay = data.maxDelay || 45;
            const delay = Math.floor(Math.random() * (maxDelay - minDelay + 1)) + minDelay;
            startTimeoutTimer(delay);
            
            // Добавляем несколько попыток клика с интервалом
            let attempts = 0;
            const maxAttempts = 3;
            const tryClick = () => {
                attempts++;
                console.log(`Attempt ${attempts} to click the button`);
                
                // Находим кнопку заново перед каждым кликом
                const buttonToClick = document.evaluate(
                    "//div[contains(@class, '_9663006')]//div[contains(@class, 'ds-icon-button') and contains(@class, 'a3b9bd76')]",
                    document,
                    null,
                    XPathResult.FIRST_ORDERED_NODE_TYPE,
                    null
                ).singleNodeValue;
                
                if (buttonToClick) {
                    console.log('Button found, trying to click...');
                    buttonToClick.click();
                    console.log('Click attempted');
                } else {
                    console.log('Button not found for clicking');
                }
                
                if (attempts < maxAttempts) {
                    setTimeout(tryClick, 1000); // Повторная попытка через 1 секунду
                }
            };
            
            setTimeout(() => {
                stopTimeoutTimer();
                tryClick(); // Начинаем попытки клика
            }, delay * 1000);
        });
        
        return true;
    }
    return false;
}

// Модифицируем observer для запуска проверки при желтом статусе
const observer = new MutationObserver(async () => {
    checkUrlChange();

    const lastResponse = document.evaluate("(//div[contains(@class, 'd7dc56a8')])[last()]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
    const loadingIcon = document.evaluate("(//div[contains(@class, 'b4e4476b')])[last()]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
    const completeIcon = document.evaluate("(//div[contains(@class, 'd7dc56a8')])[last()]//div[@class='ds-icon-button']//div[@class='ds-icon']", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
    
    if (lastResponse) {
        // 1. Immediate yellow state with verification
        if (!startTime && loadingIcon && !timeoutInterval) {
            console.log("DeepSeek Server Busy: Waiting for response...");
            changeFavicon('yellow');
            startTimer();
            waitingForResponse = true;
            
            // ЗАПУСКАЕМ ПРОВЕРКУ КНОПКИ СРАЗУ ПРИ ЖЕЛТОМ СТАТУСЕ
            startRegenerateButtonChecker();
            
            await new Promise(resolve => setTimeout(resolve, 50));
            const verifiedLoading = document.evaluate("(//div[contains(@class, 'b4e4476b')])[last()]", document, null, XPathResult.FIRST_ORDERED_NODE_TYPE, null).singleNodeValue;
            if (verifiedLoading) changeFavicon('yellow');
        }
        
        // 2. orange state and stop checking
        if (waitingForResponse && !loadingIcon) {
            const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
            console.log(`DeepSeek Server Busy: Response received in ${elapsed}s, waiting for completion...`);
            waitingForResponse = false;
            changeFavicon('orange');
            
            // ОСТАНАВЛИВАЕМ ПРОВЕРКУ КНОПКИ ПРИ ОРАНЖЕВОМ СТАТУСЕ
            if (regenerateButtonCheckInterval) {
                clearInterval(regenerateButtonCheckInterval);
                regenerateButtonCheckInterval = null;
                isCheckingForRegenerateButton = false;
                console.log('Stopped regenerate button checker (orange status) at:', new Date().toISOString());
            }
        }
        
        // 3. Delay green transition
        if (completeIcon && startTime) {
            const elapsed = ((Date.now() - startTime) / 1000).toFixed(1);
            console.log(`DeepSeek Server Busy: Operation completed. Execution time: ${elapsed}s`);
            stopTimer();

            await new Promise(resolve => setTimeout(resolve, 300));
            const responseContent = lastResponse.textContent.trim();
            checkResponseContent(responseContent, elapsed, currentTabId);
        }
    }
});

function checkAndClickRegenerateButton(shouldClick) {
    const buttonToClick = document.evaluate(
        "//div[contains(@class, '_9663006')]//div[contains(@class, 'ds-icon-button') and contains(@class, 'a3b9bd76')]",
        document,
        null,
        XPathResult.FIRST_ORDERED_NODE_TYPE,
        null
    ).singleNodeValue;

    if (buttonToClick) {
        console.log('Regenerate button FOUND at:', new Date().toISOString());
        
        // При обнаружении кнопки переходим в красный статус и останавливаем таймер
        changeFavicon('red');
        stopTimer();
        stopTimeoutTimer(); // Добавляем остановку таймера ожидания
        waitingForResponse = false;
        
        if (shouldClick) {
            //console.log('Button details:', {
            //    classes: buttonToClick.className,
            //    parentClasses: buttonToClick.parentNode.className,
            //    visible: buttonToClick.offsetParent !== null
            //});

            try {
                buttonToClick.click();
                console.log('Regenerate button CLICKED at:', new Date().toISOString());
                regenerateTimeoutActive = true;
                
                // Полный сброс состояния и запуск нового цикла мониторинга
                setTimeout(() => {
                    // Очищаем все предыдущие состояния
                    clearInterval(blinkInterval);
                    clearInterval(errorCheckInterval);
                    clearInterval(regenerateButtonCheckInterval);
                    isCheckingForRegenerateButton = false;
                    regenerateTimeoutActive = false;
                    
                    // Запускаем новый цикл мониторинга
                    startTimer();
                    changeFavicon('yellow');
                    waitingForResponse = true;
                    
                    // Запускаем проверку кнопки заново
                    startRegenerateButtonChecker();
                }, 1000);
                
                return true;
            } catch (e) {
                console.error('Error clicking button:', e);
                return false;
            }
        }
        return true;
    } else {
        //console.log('Regenerate button NOT FOUND at:', new Date().toISOString());
        return false;
    }
}

// Модифицированная функция для проверки и клика кнопки регенерации
function startRegenerateButtonChecker() {
    if (isCheckingForRegenerateButton) return;
    isCheckingForRegenerateButton = true;
    
    console.log('Starting regenerate button checker at:', new Date().toISOString());
    
    regenerateButtonCheckInterval = setInterval(() => {
        const found = checkAndClickRegenerateButton(false);
        
        if (found) {
            clearInterval(regenerateButtonCheckInterval);
            isCheckingForRegenerateButton = false;
            console.log('Found regenerate button, waiting for timeout at:', new Date().toISOString());
            
            chrome.storage.sync.get(['minDelay', 'maxDelay'], function(data) {
                const minDelay = data.minDelay || 45;
                const maxDelay = data.maxDelay || 60;
                const delay = Math.floor(Math.random() * (maxDelay - minDelay + 1)) + minDelay;
                
                // Start the timeout timer with visual indicators
                startTimeoutTimer(delay);
                
                setTimeout(() => {
                    if (checkAndClickRegenerateButton(true)) {
                        // The click was successful, the timer will be restarted in checkAndClickRegenerateButton
                        stopTimeoutTimer();
                    }
                }, delay * 1000);
            });
        }
    }, 1000);
}




// Добавляем эту функцию перед observer
function checkResponseContent(content, elapsed, currentTabId) {
    console.log('Final response content check at:', new Date().toISOString());
    setTimeout(() => changeFavicon('green'), 500);
    handleNotification(elapsed, currentTabId);
}

observer.observe(document.body, { childList: true, subtree: true });