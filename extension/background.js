chrome.action.onClicked.addListener(async (tab) => {

  try {
    // Inject a script to grab the entire HTML of the current page
    const result = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: () => document.documentElement.outerHTML,
    });

    const htmlContent = result[0].result; // Get the HTML content
    const currentUrl = tab.url; // Get the current URL

    // Create a Blob (file) from the HTML content
    const blob = new Blob([htmlContent], { type: 'text/html' });
    const formData = new FormData();
    formData.append('document', blob, 'page.html'); // Attach the file
    formData.append('chat_id', chatId); // Add chat ID
    formData.append('caption', `@shah256_bot HTML from: ${currentUrl}`); // Add caption with URL

    // Send the file to Telegram
    const url = `https://api.telegram.org/bot${botToken}/sendDocument`;
    const response = await fetch(url, {
      method: 'POST',
      body: formData,
    });

    const data = await response.json();
    if (data.ok) {
      console.log('File sent successfully!');
    } else {
      console.error('Failed to send file:', data);
    }
  } catch (error) {
    console.error('Error:', error);
  }
});
