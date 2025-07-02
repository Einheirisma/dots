// background.js
// Add at the top with other constants
const WELCOME_PAGE_URL = chrome.runtime.getURL('welcome.html');

// Add this with existing chrome.runtime.onInstalled listeners
chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    // Show welcome page first
    chrome.tabs.create({ url: WELCOME_PAGE_URL }, (welcomeTab) => {
      // Auto-refresh DeepSeek tabs after welcome page loads
      setTimeout(() => {
        chrome.tabs.query({ url: "*://chat.deepseek.com/*" }, (tabs) => {
          tabs.forEach((tab, index) => {
            if (tab.id && tab.id !== welcomeTab.id) { // Skip welcome page
              setTimeout(() => {
                chrome.tabs.reload(tab.id, { bypassCache: true });
              }, index * 1000); // 1 second between refreshes
            }
          });
        });
      }, 500); // Short delay after welcome page appears
    });
  }
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === "getCurrentTabId") {
        const currentTabId = sender.tab.id; // Directly get the tab ID from the sender
        sendResponse({ tabId: currentTabId });
        return true; // Keep for async responses
    }
});

async function tabExists(tabId) {
    return new Promise((resolve) => {
        if (typeof tabId !== 'number' || tabId <= 0) {
            console.error(`Invalid tabId: ${tabId}`);
            resolve(false);
            return;
        }

        chrome.tabs.get(tabId, (tab) => {
            if (chrome.runtime.lastError) {
                console.log(`Tab ${tabId} check failed:`, chrome.runtime.lastError.message); // Added log
                resolve(false);
            } else {
                console.log(`Tab ${tabId} exists?`, !!tab); // Added log
                resolve(!!tab);
            }
        });
    });
}

function validateNotificationData(notificationId) {
    const data = notificationMap.get(notificationId);
    if (!data || typeof data.tabId !== 'number') {
        console.error("Invalid notification data, cleaning up:", notificationId);
        notificationMap.delete(notificationId);
        chrome.notifications.clear(notificationId);
        return false;
    }
    return true;
}

const notificationMap = new Map(); // Track notifications and their tab IDs

chrome.tabs.onRemoved.addListener((closedTabId) => {
  notificationMap.forEach((value, notificationId) => {
    if (value.tabId === closedTabId) {
      // Удаляем уведомление, связанное с закрытой вкладкой
      chrome.notifications.clear(notificationId, () => {
        console.log(`Cleared notification ${notificationId} for closed tab ${closedTabId}`);
      });
      // Удаляем запись из notificationMap
      notificationMap.delete(notificationId);
    }
  });
});

const showNotification = (alertMessage, tabId) => {
  // Генерируем уникальный ID для уведомления
  const notificationId = `deepseek-notification-${Date.now()}-${tabId}`;
  
  chrome.notifications.create(notificationId, { // Явно указываем ID
    type: "basic",
    iconUrl: chrome.runtime.getURL('DeepseekServerisBusy_128.png'),
    title: "Deepseek Server Busy",
    message: alertMessage,
    requireInteraction: true
  }, () => {
    if (chrome.runtime.lastError) {
      console.error("Notification creation failed:", chrome.runtime.lastError.message);
      return;
    }
    notificationMap.set(notificationId, { tabId });
  });
};

chrome.notifications.onClicked.addListener(async (notificationId) => {
    console.log("Notification clicked:", notificationId);
    
    const data = notificationMap.get(notificationId);
    if (!data || typeof data.tabId !== 'number') {
        console.error("Invalid notification data for:", notificationId);
        notificationMap.delete(notificationId); // Очистка битых данных
        return;
    }

    const { tabId } = data;
    console.log("Attempting to focus tab:", tabId);

    try {
        // Проверка существования окна через Promise
        const window = await new Promise((resolve) => {
            chrome.windows.getCurrent((w) => resolve(w));
        });
        if (!window) {
            console.error("No active window found");
            return;
        }

        if (typeof tabId !== 'number' || tabId <= 0) {
            console.error("Invalid tabId:", tabId);
            notificationMap.delete(notificationId);
            return;
        }

        const isValidTab = await tabExists(tabId);
        if (!isValidTab) {
            console.error("Tab does not exist:", tabId);
            notificationMap.delete(notificationId);
            return;
        }

        console.log("Focusing tab ID:", tabId);
        let tab;
        try {
          tab = await chrome.tabs.get(tabId); // Проверка существования вкладки
        } catch (error) {
          console.error("Tab not found:", tabId);
          notificationMap.delete(notificationId);
          return;
        }

        // Проверка существования окна
        let windowExists;
        try {
          const window = await chrome.windows.get(tab.windowId);
          windowExists = !!window;
        } catch (error) {
          windowExists = false;
        }

        if (!windowExists) {
          console.error("Window not found:", tab.windowId);
          notificationMap.delete(notificationId);
          return;
        }

        // Активация вкладки
        await chrome.tabs.update(tabId, { active: true });

        if (chrome.runtime.lastError) {
            console.error("Tab update error:", chrome.runtime.lastError.message);
            return;
        }

        console.log("Tab activated, now focusing window...");
        try {
          await chrome.windows.update(tab.windowId, { focused: true });
        } catch (error) {
          console.error("Failed to focus window:", error);
        }

        console.log("Window focused. Clearing notification...");
        await new Promise((resolve) => {
            chrome.notifications.clear(notificationId, () => resolve());
        });

        notificationMap.delete(notificationId);
        console.log("Notification cleared. Sending reset to tab...");

        const response = await new Promise((resolve) => {
            chrome.tabs.sendMessage(tabId, { action: "resetNotificationState" }, (resp) => resolve(resp));
        });

        if (chrome.runtime.lastError) {
            console.error("Reset message error:", chrome.runtime.lastError.message);
        } else {
            console.log("Reset confirmed for tab:", tabId);
        }
    } catch (error) {
        console.error("Error handling notification click:", error);
    }
});

// Handle notification close
chrome.notifications.onClosed.addListener((notificationId) => {
    const data = notificationMap.get(notificationId);
    if (data) {
        const { tabId } = data;
        // Send a message to the content script to handle cleanup
        chrome.tabs.sendMessage(tabId, { action: "resetNotificationState" }, () => {
            notificationMap.delete(notificationId);
        });
    }
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
    if (message.action === "getCurrentTabId") {
        const currentTabId = sender.tab.id;
        sendResponse({ tabId: currentTabId });
        return true; // Для асинхронного ответа
    } 
    
    else if (message.action === "showNotification") {
        const alertMessage = message.message;
        const tabId = message.tabId;
        
        // Проверка валидности tabId
        if (typeof tabId !== 'number' || tabId <= 0) {
            console.error("Invalid tabId received:", tabId);
            return; // Не возвращаем true, так как ответ не требуется
        }
        
        showNotification(alertMessage, tabId);
    }
    
    // Можно добавить другие действия через else if
});



chrome.tabs.onRemoved.addListener((closedTabId) => {
    notificationMap.forEach((value, notificationId) => {
        if (value.tabId === closedTabId) {
            notificationMap.delete(notificationId);
            chrome.notifications.clear(notificationId);
            console.log("Cleared notification for closed tab:", closedTabId);
        }
    });
});
