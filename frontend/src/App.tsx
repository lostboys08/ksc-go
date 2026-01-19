import { useState } from 'react'
import { FileUpload } from './components/FileUpload'
import { JobComboBox } from './components/JobComboBox'
import { CostOverTimeChart } from './components/CostOverTimeChart'
import { CPIChart } from './components/CPIChart'
import { OverBudgetPhases } from './components/OverBudgetPhases'
import './App.css'

type ActiveView =
  // Project Level - Financials
  | 'budget'
  | 'commitments'
  | 'prime-contract'
  // Project Level - Controls
  | 'performance'
  | 'forecast'
  | 'change-management'
  | 'risk-register'
  // Company Level - Dashboard
  | 'wip-report'
  | 'cash-flow'
  | 'backlog-analysis'
  // Legacy views
  | 'upload'
  | 'performance-report'

function App() {
  const [activeView, setActiveView] = useState<ActiveView>('budget')
  const [jobNumber, setJobNumber] = useState('')
  const [date, setDate] = useState('')
  const [reportJobNumber, setReportJobNumber] = useState('')

  return (
    <div className="app-layout">
      <aside className="sidebar">
        <div className="sidebar-header">
          <img src="/logos/MAIN_GrayBG.svg" alt="KSC" className="sidebar-logo h-256" />
        </div>
        <nav className="sidebar-nav">
          {/* Project Level */}
          <div className="nav-section">
            <div className="nav-section-title">Project Level</div>

            <div className="nav-subsection">
              <div className="nav-subsection-title">Financials</div>
              <button
                className={`nav-item ${activeView === 'budget' ? 'active' : ''}`}
                onClick={() => setActiveView('budget')}
              >
                Budget
              </button>
              <button
                className={`nav-item ${activeView === 'commitments' ? 'active' : ''}`}
                onClick={() => setActiveView('commitments')}
              >
                Commitments (POs)
              </button>
              <button
                className={`nav-item ${activeView === 'prime-contract' ? 'active' : ''}`}
                onClick={() => setActiveView('prime-contract')}
              >
                Prime Contract (Pay Apps)
              </button>
            </div>

            <div className="nav-subsection">
              <div className="nav-subsection-title">Controls</div>
              <button
                className={`nav-item ${activeView === 'performance' ? 'active' : ''}`}
                onClick={() => setActiveView('performance')}
              >
                Performance
              </button>
              <button
                className={`nav-item ${activeView === 'forecast' ? 'active' : ''}`}
                onClick={() => setActiveView('forecast')}
              >
                Forecast
              </button>
              <button
                className={`nav-item ${activeView === 'change-management' ? 'active' : ''}`}
                onClick={() => setActiveView('change-management')}
              >
                Change Management
              </button>
              <button
                className={`nav-item ${activeView === 'risk-register' ? 'active' : ''}`}
                onClick={() => setActiveView('risk-register')}
              >
                Risk Register
              </button>
            </div>

          </div>

          {/* Company Level */}
          <div className="nav-section">
            <div className="nav-section-title">Company Level</div>

            <div className="nav-subsection">
              <div className="nav-subsection-title">Dashboard</div>
              <button
                className={`nav-item ${activeView === 'wip-report' ? 'active' : ''}`}
                onClick={() => setActiveView('wip-report')}
              >
                WIP Report
              </button>
              <button
                className={`nav-item ${activeView === 'cash-flow' ? 'active' : ''}`}
                onClick={() => setActiveView('cash-flow')}
              >
                Cash Flow Aggregate
              </button>
              <button
                className={`nav-item ${activeView === 'backlog-analysis' ? 'active' : ''}`}
                onClick={() => setActiveView('backlog-analysis')}
              >
                Backlog Analysis
              </button>
            </div>
          </div>

          {/* Utilities */}
          <div className="nav-section">
            <div className="nav-section-title">Utilities</div>
            <button
              className={`nav-item ${activeView === 'upload' ? 'active' : ''}`}
              onClick={() => setActiveView('upload')}
            >
              Upload
            </button>
          </div>
        </nav>
      </aside>

      <div className="main-content">
        {/* Project Level - Financials */}
        {activeView === 'budget' && (
          <>
            <header className="header">
              <h1>Budget</h1>
              <p>Project budget management and tracking</p>
            </header>
            <main className="main">
              <p className="coming-soon-text">Coming Soon</p>
            </main>
          </>
        )}

        {activeView === 'commitments' && (
          <>
            <header className="header">
              <h1>Commitments (POs)</h1>
              <p>Purchase orders and commitment tracking</p>
            </header>
            <main className="main">
              <p className="coming-soon-text">Coming Soon</p>
            </main>
          </>
        )}

        {activeView === 'prime-contract' && (
          <>
            <header className="header">
              <h1>Prime Contract (Pay Apps)</h1>
              <p>Payment applications and contract management</p>
            </header>
            <main className="main">
              <p className="coming-soon-text">Coming Soon</p>
            </main>
          </>
        )}

        {/* Project Level - Controls */}
        {activeView === 'performance' && (
          <>
            <header className="header">
              <h1>Performance</h1>
              <p>Track and analyze project performance metrics</p>
            </header>
            <main className="main wide">
              <div className="form-group">
                <label htmlFor="reportJob">Select Job</label>
                <JobComboBox
                  value={reportJobNumber}
                  onChange={setReportJobNumber}
                />
              </div>
              <CostOverTimeChart jobNumber={reportJobNumber} />
              <div style={{ marginTop: '24px' }}>
                <CPIChart jobNumber={reportJobNumber} />
              </div>
              <div style={{ marginTop: '24px' }}>
                <OverBudgetPhases jobNumber={reportJobNumber} />
              </div>
            </main>
          </>
        )}

        {activeView === 'forecast' && (
          <>
            <header className="header">
              <h1>Forecast</h1>
              <p>Project forecasting and projections</p>
            </header>
            <main className="main">
              <p className="coming-soon-text">Coming Soon</p>
            </main>
          </>
        )}

        {activeView === 'change-management' && (
          <>
            <header className="header">
              <h1>Change Management</h1>
              <p>Track and manage project changes</p>
            </header>
            <main className="main">
              <p className="coming-soon-text">Coming Soon</p>
            </main>
          </>
        )}

        {activeView === 'risk-register' && (
          <>
            <header className="header">
              <h1>Risk Register</h1>
              <p>Project risk identification and management</p>
            </header>
            <main className="main">
              <p className="coming-soon-text">Coming Soon</p>
            </main>
          </>
        )}

        {/* Company Level - Dashboard */}
        {activeView === 'wip-report' && (
          <>
            <header className="header">
              <h1>WIP Report</h1>
              <p>Work in Progress reporting and analysis</p>
            </header>
            <main className="main">
              <p className="coming-soon-text">Coming Soon</p>
            </main>
          </>
        )}

        {activeView === 'cash-flow' && (
          <>
            <header className="header">
              <h1>Cash Flow Aggregate</h1>
              <p>Company-wide cash flow analysis</p>
            </header>
            <main className="main">
              <p className="coming-soon-text">Coming Soon</p>
            </main>
          </>
        )}

        {activeView === 'backlog-analysis' && (
          <>
            <header className="header">
              <h1>Backlog Analysis</h1>
              <p>Company backlog and pipeline analysis</p>
            </header>
            <main className="main">
              <p className="coming-soon-text">Coming Soon</p>
            </main>
          </>
        )}

        {/* Utilities */}
        {activeView === 'upload' && (
          <>
            <header className="header">
              <h1>File Upload</h1>
              <p>Upload Excel files for payment applications</p>
            </header>
            <main className="main">
              <div className="upload-options">
                <div className="form-group">
                  <label htmlFor="jobNumber">Job Number</label>
                  <input
                    type="text"
                    id="jobNumber"
                    value={jobNumber}
                    onChange={(e) => setJobNumber(e.target.value)}
                    placeholder="e.g., JOB-001"
                  />
                </div>
                <div className="form-group">
                  <label htmlFor="date">Application Date</label>
                  <input
                    type="month"
                    id="date"
                    value={date}
                    onChange={(e) => setDate(e.target.value)}
                  />
                </div>
              </div>
              <FileUpload
                uploadType="pay-application"
                jobNumber={jobNumber}
                date={date}
                onUploadSuccess={(response) => {
                  console.log('Upload successful:', response)
                }}
                onUploadError={(error) => {
                  console.error('Upload failed:', error)
                }}
              />
            </main>
          </>
        )}

        {activeView === 'performance-report' && (
          <>
            <header className="header">
              <h1>Project Performance Report</h1>
              <p>Track and analyze project performance metrics</p>
            </header>
            <main className="main wide">
              <div className="form-group">
                <label htmlFor="reportJob">Select Job</label>
                <JobComboBox
                  value={reportJobNumber}
                  onChange={setReportJobNumber}
                />
              </div>
              <CostOverTimeChart jobNumber={reportJobNumber} />
            </main>
          </>
        )}
      </div>
    </div>
  )
}

export default App
