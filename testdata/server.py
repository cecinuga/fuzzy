from flask import Flask, request, jsonify
import time

app = Flask(__name__)

@app.route('/', methods=['POST', 'GET'])
def echo_body():
    # Attende 0.2 secondi
    time.sleep(0.05)
    
    # Legge il body ricevuto
    body = request.get_data(as_text=True)
    
    # Restituisce lo stesso body
    return body, 200, {'Content-Type': 'text/plain'}

if __name__ == '__main__':
    app.run(host='0.0.0.0', port=8080)