//popup.js
document.addEventListener('DOMContentLoaded', function () {
  const minDelayInput = document.getElementById('minDelay');
  const maxDelayInput = document.getElementById('maxDelay');
  const errorMessage = document.getElementById('errorMessage');
  const saveButton = document.getElementById('saveButton');

  // Load saved settings
  chrome.storage.sync.get(['minDelay', 'maxDelay', 'alertOption'], function(data) {
    minDelayInput.value = data.minDelay || 30;
    maxDelayInput.value = data.maxDelay || 45;
    
    const option = data.alertOption || 'inactive';
    document.querySelector(`#alert${option.charAt(0).toUpperCase() + option.slice(1)}`).checked = true;
  });

  // Save handler
  saveButton.addEventListener('click', function() {
    const minDelay = parseInt(minDelayInput.value);
    const maxDelay = parseInt(maxDelayInput.value);
    const alertOption = document.querySelector('input[name="alertOption"]:checked')?.value || 'inactive';
    if (!minDelay || !maxDelay || minDelay > maxDelay || minDelayInput.value < 0 || maxDelayInput.value < 0) {
      errorMessage.style.display = 'block';
      return;
    }

    chrome.storage.sync.set({ minDelay, maxDelay, alertOption }, () => window.close());
  });

  // Input validation
  const validateInputs = () => {
    const min = parseInt(minDelayInput.value);
    const max = parseInt(maxDelayInput.value);
    errorMessage.style.display = (min > max) ? 'block' : 'none';
  };

  minDelayInput.addEventListener('input', validateInputs);
  maxDelayInput.addEventListener('input', validateInputs);
});

