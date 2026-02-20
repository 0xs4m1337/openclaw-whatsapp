package api

import (
	"encoding/base64"
	"net/http"

	"github.com/openclaw/whatsapp/bridge"
)

type qrDataResponse struct {
	Status string `json:"status"`
	QRPNG  string `json:"qr_png,omitempty"`
	Phone  string `json:"phone,omitempty"`
}

func (s *Server) handleQRData(w http.ResponseWriter, r *http.Request) {
	status := s.Client.GetStatus()
	resp := qrDataResponse{Status: string(status)}

	if status == bridge.StatusConnected {
		resp.Phone = s.Client.GetJID()
	} else {
		qrText := s.Client.GetLatestQR()
		if qrText != "" {
			png, err := bridge.GenerateQRPNG(qrText, 512)
			if err == nil {
				resp.QRPNG = base64.StdEncoding.EncodeToString(png)
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleQRPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(qrPageHTML))
}

const qrPageHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>OpenClaw WhatsApp — Link Device</title>
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    background: #0a0a0a;
    color: #e0e0e0;
    display: flex;
    justify-content: center;
    align-items: center;
    min-height: 100vh;
  }
  .card {
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 16px;
    padding: 48px;
    text-align: center;
    max-width: 460px;
    width: 100%;
  }
  h1 { font-size: 20px; font-weight: 600; margin-bottom: 8px; }
  .subtitle { color: #888; font-size: 14px; margin-bottom: 32px; }
  #qr-container {
    width: 280px; height: 280px;
    margin: 0 auto 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #fff;
    border-radius: 12px;
  }
  #qr-container img { width: 260px; height: 260px; }
  #status {
    font-size: 14px;
    color: #888;
    margin-top: 8px;
  }
  .connected {
    color: #4ade80 !important;
    font-size: 18px !important;
    font-weight: 600;
  }
  .waiting { color: #888; font-size: 13px; }
  #phone { color: #4ade80; font-size: 14px; margin-top: 4px; }
</style>
</head>
<body>
<div class="card">
  <h1>Link WhatsApp</h1>
  <p class="subtitle">Open WhatsApp on your phone, go to Settings &gt; Linked Devices &gt; Link a Device</p>
  <div id="qr-container">
    <span class="waiting" id="loading">Loading QR code...</span>
  </div>
  <div id="status"></div>
  <div id="phone"></div>
</div>
<script>
(function() {
  var container = document.getElementById('qr-container');
  var statusEl = document.getElementById('status');
  var phoneEl = document.getElementById('phone');
  var loadingEl = document.getElementById('loading');
  var currentImg = null;

  function clearChildren(el) {
    while (el.firstChild) el.removeChild(el.firstChild);
  }

  function poll() {
    fetch('/qr/data')
      .then(function(r) { return r.json(); })
      .then(function(data) {
        if (data.status === 'connected') {
          clearChildren(container);
          var checkmark = document.createElement('span');
          checkmark.className = 'connected';
          checkmark.textContent = '\u2713';
          container.appendChild(checkmark);
          statusEl.className = 'connected';
          statusEl.textContent = 'Connected';
          phoneEl.textContent = data.phone || '';
          return;
        }
        if (data.qr_png) {
          if (loadingEl && loadingEl.parentNode) loadingEl.parentNode.removeChild(loadingEl);
          if (!currentImg) {
            currentImg = document.createElement('img');
            currentImg.setAttribute('alt', 'QR Code');
            clearChildren(container);
            container.appendChild(currentImg);
          }
          currentImg.setAttribute('src', 'data:image/png;base64,' + data.qr_png);
          statusEl.textContent = 'Scan this QR code with WhatsApp';
          statusEl.className = '';
        } else {
          statusEl.textContent = 'Waiting for QR code...';
          statusEl.className = '';
        }
      })
      .catch(function() {
        statusEl.textContent = 'Connection error, retrying...';
      });
  }

  poll();
  setInterval(poll, 3000);
})();
</script>
</body>
</html>`
