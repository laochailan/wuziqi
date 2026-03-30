var board;
var gameState = States.DISCONNECTED;
var strokeColor;
var socket = null;

const PlayerMessageTypes = Object.freeze({
  MOVE: 0,
  RESIGN: 1,
  RESTART: 2,
})

const HostMessageTypes = Object.freeze({
  STATUS_UPDATE: 0,
  WAIT_FOR_PLAYER_JOIN: 1,
  REQUEST_TURN: 2,
  GAME_OVER: 3,
})

const States = Object.freeze({
  REQUEST_TURN: 0,
  AWAIT_OPPONENT_TURN: 1,
  GAME_OVER: 2,
  DISCONNECTED: 3,
})

function boardPlayerOfTurn(board, turn) {
  return turn % 2;
}

function roundPx(x) {
  return Math.round(x);
}

function socketListener(event) {
  msg = JSON.parse(event.data);

  var messageBox = document.getElementById("message-box");
  var resignLink = document.getElementById("resign-link");

  if(msg.Type == HostMessageTypes.REQUEST_TURN) {
    messageBox.textContent = "Your turn!";
    resignLink.style.visibility = "visible";
    gameState = States.REQUEST_TURN;
  } else if(msg.Type == HostMessageTypes.STATUS_UPDATE) {
    messageBox.textContent = "Waiting for opponent turn.";
    resignLink.style.visibility = "hidden";
    gameState = States.AWAIT_OPPONENT_TURN;
  } else if(msg.Type == HostMessageTypes.WAIT_FOR_PLAYER_JOIN) {
    messageBox.textContent = "Waiting for opponent to connect.";
  } else if(msg.Type == HostMessageTypes.GAME_OVER) {
    if(msg.Board.Winner == msg.Player) {
      messageBox.innerHTML = "You win!";
    } else if(msg.Board.Winner == !msg.Player) {
      messageBox.innerHTML = "You lose!";
    }

    if(boardPlayerOfTurn(msg.Board, msg.Board.Turn) == msg.Player) {
      const restartLink = ' <a href="#" onclick="restartGame()">New Game?</a>';
      messageBox.innerHTML += restartLink;
    }

    resignLink.style.visibility = "hidden";
    gameState = States.GAME_OVER;
  }

  if(msg.Board != null) {
    board = msg.Board;

    if(board.Turn > 1) {
      var shareLinkBox = document.getElementById("share-link-box");
      if(shareLinkBox) {
        shareLinkBox.classList.add("deactivated");
      }
    }
    if(board.Winner != -1) {
      gameState = States.GAME_OVER;
    }
  }

  drawCanvas();
}

function onload() {
  const canvas = document.getElementById("boardCanvas");
  strokeColor = window.getComputedStyle(document.body).getPropertyValue("--text-dark");

  socket = new WebSocket(socketURL);
  socket.addEventListener("message", socketListener);
  socket.addEventListener("close", (event) => {
    gameState = States.DISCONNECTED;
    var messageBox = document.getElementById("message-box");
    messageBox.innerHTML = "Disconnected. <a href=\"../../\">New Game?</a>";
  });

  const observer = new ResizeObserver((entries) => {
    for(const entry of entries) {
      const { width, height } = entry.contentRect;
      const scale = window.devicePixelRatio;
      canvas.width = width * scale;
      canvas.height = height * scale;
      drawCanvas();
    }
  })
  observer.observe(canvas);

  function getEventTile(event) {
    const style = window.getComputedStyle(canvas)
    const width = parseInt(style.width, 10);
    const height = parseInt(style.height, 10);
    const borderWidthX = parseInt(style.borderLeftWidth, 10);
    const borderWidthY = parseInt(style.borderTopWidth, 10);
    const x = Math.floor((event.offsetX+borderWidthX) / ((width-borderWidthX*2) / boardSize));
    const y = Math.floor((event.offsetY+borderWidthY) / ((height-borderWidthY*2) / boardSize));
    return [x, y]
  }

  drawCanvas();
  canvas.addEventListener("click", (event) => {
    const [x, y] = getEventTile(event)

    if(gameState == States.REQUEST_TURN) {
      socket.send(JSON.stringify({
        Type: PlayerMessageTypes.MOVE,
        Move: {
          X: x,
          Y: y,
        }
      }));
      gameState = States.AWAIT_OPPONENT_TURN;
    }
  });

  var clickable = false;
  canvas.addEventListener("mousemove", (event) => {
    const [x, y] = getEventTile(event)

    var newClickable = gameState == States.REQUEST_TURN && board.Tiles[y*boardSize + x] == 0;
    if(newClickable != clickable) {
      if(newClickable) {
        canvas.classList.add("hover-active");
      } else {
        canvas.classList.remove("hover-active");
      }
      clickable = newClickable;
    }
  });
}

function resign() {
  if(gameState == States.REQUEST_TURN) {
    socket.send(JSON.stringify({
      Type: PlayerMessageTypes.RESIGN,
      Move: {}
    }));
  }
}

function restartGame() {
  if(gameState == States.GAME_OVER) {
    socket.send(JSON.stringify({
      Type: PlayerMessageTypes.RESTART,
      Move: {},
    }));
  }
}

function drawX(ctx, x, y, w, h) {
  ctx.moveTo(x+0.15*w,y+0.15*h);
  ctx.lineTo(x+0.85*w,y+0.85*h);
  ctx.moveTo(x+0.85*w,y+0.15*h);
  ctx.lineTo(x+0.15*w,y+0.85*h);
  ctx.lineWidth = 4;
  ctx.strokeStyle = strokeColor;
  ctx.stroke();
}

function drawO(ctx, x, y, w, h) {
  ctx.moveTo(x + w * 0.85, y + h / 2);
  ctx.ellipse(x + w / 2, y + h / 2, 0.7 * w / 2, 0.7*h / 2, 0 , 0, 2 * Math.PI);
  ctx.lineWidth = 4;
  ctx.strokeStyle = strokeColor;
  ctx.stroke();
}

function drawN(ctx, n, x, y, w, h) {
  ctx.moveTo(x + w / 2, y + h / 2);
  ctx.textAlign = "center";
  ctx.textBaseline = "middle";

  ctx.fillStyle = "white";
  ctx.font = "bold " + (0.7*h).toString() + "px Nunito";
  ctx.fillText(n.toString(), x + w / 2, y + h / 2 * 1.1);
  ctx.fillStyle = strokeColor;
}

function drawCanvas() {
  const canvas = document.getElementById("boardCanvas");
  var ctx = canvas.getContext("2d");
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  const dx = canvas.width / boardSize;
  const dy = canvas.height / boardSize;

  const style = window.getComputedStyle(canvas);
  const scale = window.devicePixelRatio;
  const borderWidthX = parseInt(style.borderLeftWidth, 10) + parseInt(style.borderRightWidth, 10);
  const borderWidthY = parseInt(style.borderBottomWidth, 10) + parseInt(style.borderTopWidth, 10);
  canvas.width = (parseInt(style.width, 10) - borderWidthX) * scale;
  canvas.height = (parseInt(style.height, 10) - borderWidthY) * scale;

  if(board) {
    if(gameState == States.GAME_OVER) {
      for(var x = 0; x < boardSize; x++) {
        for(var y = 0; y < boardSize; y++) {
          var tile = board.Tiles[y*boardSize + x]
          if(tile != 0) {
            var color = "#a74f42aa";
            if(boardPlayerOfTurn(board, tile) == board.Winner) {
              var color = "#008198aa";
            }
            if((tile + board.FirstUseX)%2 == 0) {
              drawX(ctx, x*dx, y*dy, dx, dy);
            } else {
              drawO(ctx, x*dx, y*dy, dx, dy);
            }

            ctx.beginPath();
            ctx.fillStyle = color;
            ctx.rect(x*dx, y*dy, dx, dy);
            ctx.fill();
            ctx.fillStyle = strokeColor;
            drawN(ctx, tile, x*dx, y*dy, dx, dy);
          }
        }
      }
    } else {
      for(var x = 0; x < boardSize; x++) {
        for(var y = 0; y < boardSize; y++) {
          var tile = board.Tiles[y*boardSize + x]
          if(tile != 0) {
            if((tile + board.FirstUseX)%2 == 0) {
              drawX(ctx, x*dx, y*dy, dx, dy);
            } else {
              drawO(ctx, x*dx, y*dy, dx, dy);
            }
          }
        }
      }
    }
  }
  ctx.beginPath();
  for(var x = 1; x < boardSize; x++) {
    ctx.moveTo(roundPx(x*dx),0);
    ctx.lineTo(roundPx(x*dx),canvas.height);
    ctx.moveTo(0,roundPx(x*dy));
    ctx.lineTo(canvas.width,roundPx(x*dy));
  }
  ctx.lineWidth = 2;
  ctx.strokeStyle = strokeColor;
  ctx.stroke();
}

function copyClipboard(link) {
  navigator.clipboard.writeText(link);
  var symbol = document.getElementById("share-link-box").getElementsByClassName("copy-symbol")[0];
  symbol.src = baseURL + "/assets/check.svg";
  setTimeout(function() {
    symbol.src = baseURL + "/assets/copy.svg";
  }, 1000);
}
