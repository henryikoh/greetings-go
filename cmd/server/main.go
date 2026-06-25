package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"example.com/greetings"
)

// entry is one recorded greeting, shared with all visitors.
type entry struct {
	Name      string `json:"name"`
	Message   string `json:"message"`
	TimeOfDay string `json:"timeOfDay"`
	At        int64  `json:"at"` // unix milliseconds
}

// In-memory, server-side history. Shared across ALL visitors — but
// volatile: it's wiped whenever the process restarts (e.g. every deploy).
var (
	historyMu sync.Mutex
	history   []entry // newest first
)

func recordGreeting(name, message, tod string) {
	historyMu.Lock()
	defer historyMu.Unlock()
	history = append([]entry{{Name: name, Message: message, TimeOfDay: tod, At: time.Now().UnixMilli()}}, history...)
	if len(history) > 50 { // keep the most recent 50
		history = history[:50]
	}
}

func snapshotHistory() []entry {
	historyMu.Lock()
	defer historyMu.Unlock()
	out := make([]entry, len(history))
	copy(out, history)
	return out
}

func main() {
	// BASE_PATH lets the app be mounted under a sub-path (e.g. /greetings)
	// or at the root (empty). No trailing slash.
	base := strings.TrimRight(os.Getenv("BASE_PATH"), "/")

	mux := http.NewServeMux()
	mux.HandleFunc(base+"/", handleIndex(base))
	mux.HandleFunc(base+"/api/greet", handleGreet)
	mux.HandleFunc(base+"/api/history", handleHistory)
	mux.HandleFunc(base+"/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	addr := "127.0.0.1:8080"
	log.Printf("greetings server listening on %s (base path %q)", addr, base)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// handleGreet returns a time-aware greeting as JSON and records it.
// ?name=<name>&hour=<0-23, the caller's local hour>
func handleGreet(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	hour := time.Now().Hour() // fall back to server time
	if h := r.URL.Query().Get("hour"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed >= 0 && parsed <= 23 {
			hour = parsed
		}
	}

	w.Header().Set("Content-Type", "application/json")
	msg, part, err := greetings.Greet(name, hour)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	recordGreeting(strings.TrimSpace(name), msg, part)
	json.NewEncoder(w).Encode(map[string]string{"message": msg, "timeOfDay": part})
}

// handleHistory returns the shared, server-side greeting history.
func handleHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshotHistory())
}

// handleIndex serves the browser UI, with the base path baked into the page
// so the front-end fetches from the right place.
func handleIndex(base string) http.HandlerFunc {
	page := strings.ReplaceAll(indexHTML, "{{BASE}}", base)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != base+"/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(page))
	}
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Greetings · Henry's Lab</title>
<style>
  *{margin:0;padding:0;box-sizing:border-box}
  body{font-family:-apple-system,system-ui,sans-serif;min-height:100vh;display:flex;align-items:center;justify-content:center;
    background:radial-gradient(circle at 30% 20%,#26282b,#131315 65%);color:#ececec;padding:2rem}
  .card{width:100%;max-width:460px;text-align:center}
  h1{font-size:2rem;margin-bottom:1.6rem;background:linear-gradient(90deg,#fff,#6ee7b7);-webkit-background-clip:text;-webkit-text-fill-color:transparent}
  .row{display:flex;gap:.5rem;margin-bottom:1.2rem}
  input{flex:1;padding:.8rem 1rem;border-radius:10px;border:1px solid rgba(255,255,255,.12);
    background:rgba(255,255,255,.05);color:#fff;font-size:1rem;outline:none}
  input:focus{border-color:#10b981}
  button{padding:.8rem 1.3rem;border:0;border-radius:10px;background:#10b981;color:#06281d;font-size:1rem;font-weight:600;cursor:pointer}
  button:hover{background:#0e9d72}
  .result{min-height:2.4rem;font-size:1.25rem;font-weight:600;margin-bottom:.4rem;transition:opacity .2s}
  .tod{font-size:.75rem;color:#7a7d82;text-transform:uppercase;letter-spacing:.12em;margin-bottom:1.8rem;min-height:1rem}
  .recent-label{font-size:.7rem;color:#6c6f74;text-transform:uppercase;letter-spacing:.12em;margin-bottom:.6rem;text-align:left}
  .history{display:flex;flex-direction:column;gap:.5rem}
  .item{text-align:left;background:rgba(255,255,255,.04);border:1px solid rgba(255,255,255,.07);
    padding:.6rem .8rem;border-radius:10px}
  .item .msg{font-size:.95rem;color:#dcdfe2}
  .item .meta{font-size:.62rem;color:#7a7d82;text-transform:uppercase;letter-spacing:.1em;margin-top:.25rem}
  footer{margin-top:2.5rem;font-size:.72rem;color:#6c6f74;font-family:ui-monospace,monospace}
</style>
</head>
<body>
<div class="card">
  <h1>Greetings 👋</h1>
  <div class="row">
    <input id="name" placeholder="What's your name?" autocomplete="off">
    <button id="go">Greet</button>
  </div>
  <div class="result" id="result"></div>
  <div class="tod" id="tod"></div>
  <div class="recent-label" id="recentLabel" style="display:none">Previous visitors</div>
  <div class="history" id="history"></div>
  <footer>self-hosted in Henry's lab</footer>
</div>
<script>
  var BASE='{{BASE}}';
  var input=document.getElementById('name');
  var resultEl=document.getElementById('result');
  var todEl=document.getElementById('tod');
  var histEl=document.getElementById('history');
  var recentLabel=document.getElementById('recentLabel');

  function timeAgo(ts){
    var s=Math.floor((Date.now()-ts)/1000);
    if(s<60)return 'just now';
    var m=Math.floor(s/60); if(m<60)return m+'m ago';
    var h=Math.floor(m/60); if(h<24)return h+'h ago';
    return Math.floor(h/24)+'d ago';
  }
  function renderHistory(items){
    histEl.innerHTML='';
    recentLabel.style.display=items.length?'block':'none';
    items.forEach(function(i){
      var row=document.createElement('div'); row.className='item';
      var msg=document.createElement('div'); msg.className='msg'; msg.textContent=i.message;
      var meta=document.createElement('div'); meta.className='meta';
      meta.textContent=(i.timeOfDay?i.timeOfDay+' · ':'')+timeAgo(i.at);
      row.appendChild(msg); row.appendChild(meta); histEl.appendChild(row);
    });
  }
  function loadHistory(){
    fetch(BASE+'/api/history').then(function(r){return r.json()})
      .then(function(items){renderHistory(items||[])}).catch(function(){});
  }
  function greet(name){
    name=(name||'').trim();
    if(!name){resultEl.textContent='Please enter a name';todEl.textContent='';return}
    var hour=new Date().getHours();
    resultEl.style.opacity='.35';
    fetch(BASE+'/api/greet?name='+encodeURIComponent(name)+'&hour='+hour)
      .then(function(r){return r.json()})
      .then(function(data){
        resultEl.style.opacity='1';
        if(data.error){resultEl.textContent='⚠️ '+data.error;todEl.textContent='';return}
        resultEl.textContent=data.message;
        todEl.textContent=data.timeOfDay;
        loadHistory();
      })
      .catch(function(){resultEl.style.opacity='1';resultEl.textContent='⚠️ network error'});
  }
  document.getElementById('go').onclick=function(){greet(input.value)};
  input.addEventListener('keydown',function(e){if(e.key==='Enter')greet(input.value)});
  loadHistory();
  input.focus();
</script>
</body>
</html>`
