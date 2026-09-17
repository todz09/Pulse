import {useState, useEffect} from 'react';
import './App.css';

const API_URL = 'http://localhost:8080';

function App(){
  const [monitors, setMonitors] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchMonitors();
  },[])

  async function fetchMonitors() {
    try {
      const response = await fetch(`${API_URL}/api/monitors`);
      if (!response.ok) {
        throw new Error(`API returned ${response.status}`);
      }
      const data = await response.json();
      setMonitors(data);
      setError(null);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }

  if(loading) {
    return <div className = "app"><p>Loading Monitors...</p></div>
  }
  if (error){
    return <div className="app"><p>Error: {error}</p></div>
  }

  return(
    <div className="app">
      <h1>Pulse</h1>
      <p>Uptime Monitoring dasboard</p>

      <div className="monitor-list">
        {monitors.map((monitor) => (
          <div key={monitor.id} className="monitor-card">
            <h2>{monitor.name}</h2>
            <p>{monitor.url}</p>
            <p>Status: {monitor.up ? 'UP' : 'Down'}</p>
            <p>Response time : {monitor.response_time_ms}</p>

          </div>
        ))}

      </div>

    </div>
  )
}

export default App;