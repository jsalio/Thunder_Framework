use futures_util::SinkExt;
use tokio::net::TcpListener;
use tokio::sync::broadcast;
use tokio_tungstenite::accept_async;
use tokio_tungstenite::tungstenite::Message;

/// Starts the WebSocket server on `127.0.0.1:<port>`.
/// Returns a `broadcast::Sender` — calling `.send("reload")` signals all connected browsers.
///
/// Each browser tab gets its own task subscribed to the broadcast channel.
/// Lagged receivers silently drop stale signals — reloading is idempotent.
pub async fn start(port: u16) -> broadcast::Sender<()> {
    let (tx, _rx) = broadcast::channel::<()>(16);
    let tx_clone = tx.clone();

    let addr = format!("127.0.0.1:{port}");
    let listener = TcpListener::bind(&addr)
        .await
        .unwrap_or_else(|e| panic!("failed to bind WebSocket server on {addr}: {e}"));

    tracing::info!("live-reload WebSocket listening on ws://{addr}");

    tokio::spawn(async move {
        loop {
            match listener.accept().await {
                Ok((stream, peer)) => {
                    tracing::debug!("browser connected: {peer}");
                    let rx = tx_clone.subscribe();
                    tokio::spawn(handle_client(stream, rx));
                }
                Err(e) => {
                    tracing::warn!("WebSocket accept error: {e}");
                }
            }
        }
    });

    tx
}

/// Handles a single browser connection.
/// Subscribes to the broadcast channel and forwards every signal as a WS text message.
/// Exits when the browser disconnects or the broadcast channel closes.
async fn handle_client(
    stream: tokio::net::TcpStream,
    mut rx: broadcast::Receiver<()>,
) {
    let ws = match accept_async(stream).await {
        Ok(ws) => ws,
        Err(e) => {
            tracing::debug!("WebSocket handshake failed: {e}");
            return;
        }
    };

    let (mut sink, _stream) = futures_util::StreamExt::split(ws);

    loop {
        match rx.recv().await {
            Ok(()) => {
                if sink.send(Message::Text("reload".into())).await.is_err() {
                    // Browser disconnected — exit the task cleanly.
                    break;
                }
            }
            Err(broadcast::error::RecvError::Lagged(_)) => {
                // Missed some signals — a single reload still suffices.
                if sink.send(Message::Text("reload".into())).await.is_err() {
                    break;
                }
            }
            Err(broadcast::error::RecvError::Closed) => {
                // Watcher is shutting down.
                break;
            }
        }
    }
}
