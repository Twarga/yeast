from flask import Flask
import os

app = Flask(__name__)
node = os.environ.get("WEB_NODE", "unknown")

@app.route("/")
def index():
    return f"<h1>Hello from {node}</h1><p>Node: {node}</p>"

@app.route("/health")
def health():
    return {"status": "ok", "node": node}

if __name__ == "__main__":
    port = int(os.environ.get("WEB_PORT", 8080))
    app.run(host="0.0.0.0", port=port)
