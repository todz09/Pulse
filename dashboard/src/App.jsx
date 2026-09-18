import {useState, useEffect} from 'react';
import MonitorCard from './MonitorCard';
import './App.css';

const API_URL = 'http://localhost:8080';

function App(){
  const [monitors, setMonitors] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    fetchMonitors();
    const interval = setInterval(fetchMonitors, 30000);
    return () => clearInterval(interval);
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
    return <div className = "app-status"><p>Loading Monitors...</p></div>
  }
  if (error){
    return <div className="app-status"><p>Error: {error}</p></div>
  }

  const upCount = monitors.filter((m) => m.up).length;

  return(
    <div className="app">
      <header className="app-header">
      <h1>Pulse</h1>
      <p>{upCount} of {monitors.length} monitors Operational</p>
      </header>

      <div className="monitor-list">
        {monitors.map((monitor) => (
          <MonitorCard key={monitor.id} monitor={monitor} />
        ))}
      </div>
    </div>
  );
}

export default App;