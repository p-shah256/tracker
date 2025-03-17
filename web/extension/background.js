chrome.action.onClicked.addListener(async (tab) => {
  try {
    const config = await fetch(chrome.runtime.getURL('config.json')).then((response) => response.json());
    const discordWebhookUrl = config.discordWebhookUrl;
    const botId = config.botId; // Add the bot's ID to your config file

    const result = await chrome.scripting.executeScript({
      target: { tabId: tab.id },
      func: () => document.documentElement.outerHTML,
    });

    const htmlContent = result[0].result;
    const currentUrl = tab.url;

    const blob = new Blob([htmlContent], { type: 'text/html' });
    const formData = new FormData();
    formData.append('file', blob, 'page.html');

    const payload = {
      content: `<@${botId}> ${currentUrl}`,
    };
    formData.append('payload_json', JSON.stringify(payload));

    const response = await fetch(discordWebhookUrl, {
      method: 'POST',
      body: formData,
    });

    if (response.ok) {
      console.log('File sent successfully to Discord!');
    } else {
      const errorData = await response.json();
      console.error('Failed to send file to Discord:', errorData);
    }
  } catch (error) {
    console.error('Error:', error);
  }
});
