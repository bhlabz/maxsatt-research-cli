import requests
from credentials import token, chat_id

def send_telegram_message(message, parse_mode=None):
    """
    Sends a message to Telegram.
    
    :param token: Your bot's token.
    :param chat_id: The chat ID where the message will be sent.
    :param message: The message text.
    :param parse_mode: Optional. Can be "MarkdownV2" or "HTML" for formatted messages.
    """

    message = message.replace("!", "\\!")
    url = f"https://api.telegram.org/bot{token}/sendMessage"
    payload = {
        "chat_id": chat_id,
        "text": message,
    }
    if parse_mode:
        payload["parse_mode"] = parse_mode

    response = requests.post(url, json=payload)
    print(f"Telegram Response Status Code: {response.status_code}")
    print(f"Telegram Response Content: {response.content}")
    return response.json()

def send_error_message(exception, message):
    """
    Sends an error message with Exception details to Telegram using Markdown.
    
    :param token: Your bot's token.
    :param chat_id: The chat ID where the message will be sent.
    :param exception: The Exception object.
    """
 
    error_message = (
        f"*🚨 Error Occurred!* \n\n"
        f"*Type:* `{type(exception).__name__}`\n"
        f"*Message:* `{message}`\n"
        f"*Error Message:* `{str(exception)}`"
    )
    send_telegram_message(error_message, parse_mode="MarkdownV2")

def send_success_message(message):
    """
    Sends a success message to Telegram using Markdown.
    
    :param token: Your bot's token.
    :param chat_id: The chat ID where the message will be sent.
    :param message: The success message text.
    """
    success_message = f"*✅ Success!* \n\n{message}"
    send_telegram_message(success_message, parse_mode="MarkdownV2")


def send_warn_message(message):
    """
    Sends a warning message to Telegram using Markdown.
    
    :param token: Your bot's token.
    :param chat_id: The chat ID where the message will be sent.
    :param message: The warning message text.
    """
    warn_message = f"*⚠️ Warning!* \n\n{message}"
    send_telegram_message(warn_message, parse_mode="MarkdownV2")