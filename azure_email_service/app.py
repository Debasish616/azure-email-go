from flask import Flask, request, jsonify
from azure.communication.email import EmailClient
import os

app = Flask(__name__)

@app.route("/")
def index():
    return "<h1>Hello!</h1>"

@app.route('/send-email', methods=['POST'])
def send_email():
    try:
        data = request.get_json()
        email_address = data.get('email')
        subject = data.get('subject')
        plain_text = data.get('plainText')
        html_content = data.get('htmlContent', None)

        if not email_address or not subject or not plain_text:
            return jsonify({"error": "Email address, subject, and plain text content are required"}), 400

        connection_string = os.getenv('AZURE_CONNECTION_STRING')
        sender_address = os.getenv('SENDER_ADDRESS')
        client = EmailClient.from_connection_string(connection_string)

        message = {
            "senderAddress": sender_address,
            "recipients": {
                "to": [{"address": email_address}],
            },
            "content": {
                "subject": subject,
                "plainText": plain_text,
            }
        }

        if html_content:
            message["content"]["html"] = html_content

        poller = client.begin_send(message)
        result = poller.result()

        return jsonify({"message": "Email sent successfully"}), 200

    except Exception as ex:
        return jsonify({"error": str(ex)}), 500

if __name__ == '__main__':
    app.run(host="0.0.0.0", port=8005)
