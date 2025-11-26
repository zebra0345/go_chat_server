let socket = null;
const nicknameSpan = document.getElementById("nickname");
const logDiv = document.getElementById("log");
const connectBtn = document.getElementById("connectBtn");
const disconnectBtn = document.getElementById("disconnectBtn");
const chatInput = document.getElementById("chatInput");
const sendBtn = document.getElementById("sendBtn");

function log(message) {
  const time = new Date().toLocaleTimeString();
  logDiv.textContent += `[${time}] ${message}\n`;
  logDiv.scrollTop = logDiv.scrollHeight;
}

function connect() {
  if (socket && socket.readyState === WebSocket.OPEN) {
    log("이미 연결되어 있습니다.");
    return;
  }

  const wsProtocol = location.protocol === "https:" ? "wss" : "ws";
  const wsUrl = `${wsProtocol}://${location.host}/ws`;
  socket = new WebSocket(wsUrl);

  socket.onopen = () => {
    log("서버와 연결됨");
    connectBtn.disabled = true;
    disconnectBtn.disabled = false;
    sendBtn.disabled = false;
  };

  socket.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);
      if (data.type === "welcome") {
        nicknameSpan.textContent = data.nickname;
        log(`서버로부터 닉네임 할당: ${data.nickname}`);
      } else if (data.type === "system") {
        log(`SYSTEM: ${data.message}`);
      } else if (data.type === "chat") {
        log(`${data.nickname}: ${data.message}`);
      } else {
        log(`UNKNOWN MESSAGE: ${event.data}`);
      }
    } catch (e) {
      log(`RAW: ${event.data}`);
    }
  };

  socket.onclose = () => {
    log("서버와의 연결이 종료되었습니다.");
    nicknameSpan.textContent = "(연결 안 됨)";
    connectBtn.disabled = false;
    disconnectBtn.disabled = true;
    sendBtn.disabled = true;
  };

  socket.onerror = (err) => {
    log("에러 발생: " + err.message);
  };
}

function disconnect() {
  if (socket) {
    socket.close();
    socket = null;
  }
}

function sendChat() {
  if (!socket || socket.readyState !== WebSocket.OPEN) {
    log("서버에 연결되어 있지 않습니다.");
    return;
  }

  const text = chatInput.value.trim();
  if (!text) return;

  // 서버는 텍스트 그대로를 채팅 메시지로 처리한다.
  socket.send(text);
  chatInput.value = "";
}

connectBtn.addEventListener("click", connect);
disconnectBtn.addEventListener("click", disconnect);
sendBtn.addEventListener("click", sendChat);

// Enter 키로 전송
chatInput.addEventListener("keydown", (e) => {
  if (e.key === "Enter") {
    sendChat();
  }
});

// 자동 연결하고 싶으면 아래 주석 해제
// connect();
