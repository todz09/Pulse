function MonitorCard({monitor}){
    const isUp = monitor.up;

    return (
        <div className={`monitor-card $ {isUp ? 'status-up' : 'status-down'}`}>
            <div className = "monitor-header">
                <span className={`status-dot $ {isUp ? 'dot-up' : 'dot-down'}`}></span>
                <h2> {monitor.name}</h2>
            </div>
            <p className = "monitor-url"> {monitor.url}</p>
            <div className="monitor-stats">
                <div className="stat">
                    <span className="stat-label"> Status</span>
                    <span className="stat-value">{isUp ? 'Operational':'Down'}</span>
                </div>
                <div className="stat">
                    <span className="stat-label"> Status code</span>
                    <span className="stat-value"> {monitor.status_code ?? '-'}</span>
                </div>
            </div>
        </div>
    );
}

export default MonitorCard;