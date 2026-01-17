import { useState } from 'react'
import { FileUpload } from './components/FileUpload'
import { JobComboBox } from './components/JobComboBox'
import { CostOverTimeChart } from './components/CostOverTimeChart'
import './App.css'

type ActiveView = 'upload' | 'performance-report' | 'forecast'

function App() {
  const [activeView, setActiveView] = useState<ActiveView>('upload')
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
          <button
            className={`nav-item ${activeView === 'upload' ? 'active' : ''}`}
            onClick={() => setActiveView('upload')}
          >
            Upload
          </button>
          <button
            className={`nav-item ${activeView === 'performance-report' ? 'active' : ''}`}
            onClick={() => setActiveView('performance-report')}
          >
            Performance Report
          </button>
          <button
            className={`nav-item ${activeView === 'forecast' ? 'active' : ''}`}
            onClick={() => setActiveView('forecast')}
          >
            Forecast
          </button>
        </nav>
      </aside>

      <div className="main-content">
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

        {activeView === 'forecast' && (
          <>
            <header className="header">
              <h1>Forecast</h1>
            </header>

            <main className="main">
              <p>Coming soon</p>
            </main>
          </>
        )}
      </div>
    </div>
  )
}

export default App
