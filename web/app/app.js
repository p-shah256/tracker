// Global state
const state = {
    extractedSkills: null,
    resume: null,
    scoredResume: null,
    transformedResume: null,
    minScore: 7
};

// DOM elements
const elements = {
    // Navigation
    stepNav: document.getElementById('step-nav'),
    
    // Step 1: Job Description
    jobDescText: document.getElementById('jobDescText'),
    jobDescFile: document.getElementById('jobDescFile'),
    extractBtn: document.getElementById('extractBtn'),
    
    // Step 2: Resume
    resumeText: document.getElementById('resumeText'),
    resumeFile: document.getElementById('resumeFile'),
    matchBtn: document.getElementById('matchBtn'),
    
    // Step 3: Matching
    extractedSkills: document.getElementById('extractedSkills'),
    matchingScores: document.getElementById('matchingScores'),
    minScore: document.getElementById('minScore'),
    minScoreValue: document.getElementById('minScoreValue'),
    transformBtn: document.getElementById('transformBtn'),
    
    // Step 4: Transformation
    transformedEntries: document.getElementById('transformedEntries'),
    downloadBtn: document.getElementById('downloadBtn'),
    
    // Loading overlay
    loadingOverlay: document.getElementById('loadingOverlay'),
    loadingMessage: document.getElementById('loadingMessage')
};

// Initialize the application
function init() {
    // Set up event listeners
    elements.extractBtn.addEventListener('click', handleExtract);
    elements.matchBtn.addEventListener('click', handleMatch);
    elements.transformBtn.addEventListener('click', handleTransform);
    elements.downloadBtn.addEventListener('click', handleDownload);
    elements.minScore.addEventListener('input', updateMinScoreValue);
    
    // Set up tab navigation
    document.querySelectorAll('#step-nav a').forEach(tab => {
        tab.addEventListener('click', (e) => {
            e.preventDefault();
            document.querySelectorAll('#step-nav a').forEach(t => t.classList.remove('active'));
            tab.classList.add('active');
            
            const targetId = tab.getAttribute('href');
            document.querySelectorAll('.tab-pane').forEach(pane => pane.classList.remove('show', 'active'));
            document.querySelector(targetId).classList.add('show', 'active');
        });
    });
}

// Show loading overlay
function showLoading(message = 'Processing...') {
    elements.loadingMessage.textContent = message;
    elements.loadingOverlay.classList.remove('d-none');
}

// Hide loading overlay
function hideLoading() {
    elements.loadingOverlay.classList.add('d-none');
}

// Update min score value display
function updateMinScoreValue() {
    state.minScore = parseInt(elements.minScore.value);
    elements.minScoreValue.textContent = state.minScore;
}

// Navigate to a specific step
function navigateToStep(stepNumber) {
    const stepLink = document.querySelector(`#step-nav a[href="#step${stepNumber}"]`);
    if (stepLink) {
        stepLink.click();
    }
}

// Handle extract button click
async function handleExtract() {
    try {
        showLoading('Extracting skills from job description...');
        
        // Create form data
        const formData = new FormData();
        
        // Add job description text or file
        if (elements.jobDescText.value.trim()) {
            formData.append('jobDescText', elements.jobDescText.value.trim());
        } else if (elements.jobDescFile.files.length > 0) {
            formData.append('jobDescFile', elements.jobDescFile.files[0]);
        } else {
            alert('Please provide a job description');
            hideLoading();
            return;
        }
        
        // Call API
        const response = await fetch('/api/extract', {
            method: 'POST',
            body: formData
        });
        
        if (!response.ok) {
            throw new Error(`API error: ${response.status}`);
        }
        
        // Parse response
        state.extractedSkills = await response.json();
        
        // Display extracted skills
        displayExtractedSkills();
        
        // Enable match button
        elements.matchBtn.disabled = false;
        
        // Navigate to next step
        navigateToStep(2);
    } catch (error) {
        console.error('Extract error:', error);
        alert(`Error extracting skills: ${error.message}`);
    } finally {
        hideLoading();
    }
}

// Display extracted skills
function displayExtractedSkills() {
    if (!state.extractedSkills) return;
    
    const { required_skills, nice_to_have_skills, company_info } = state.extractedSkills;
    
    let html = `
        <div class="mb-3">
            <h6>Company: ${company_info.name || 'N/A'}</h6>
            <h6>Position: ${company_info.position || 'N/A'}</h6>
            <h6>Level: ${company_info.level || 'N/A'}</h6>
        </div>
        <div class="mb-3">
            <h6>Required Skills:</h6>
            <div>
    `;
    
    required_skills.forEach(skill => {
        html += `<span class="skill-tag required">${skill.name}</span>`;
    });
    
    html += `
            </div>
        </div>
        <div class="mb-3">
            <h6>Nice-to-Have Skills:</h6>
            <div>
    `;
    
    nice_to_have_skills.forEach(skill => {
        html += `<span class="skill-tag nice-to-have">${skill.name}</span>`;
    });
    
    html += `
            </div>
        </div>
    `;
    
    elements.extractedSkills.innerHTML = html;
}

// Handle match button click
async function handleMatch() {
    try {
        showLoading('Matching resume against job requirements...');
        
        // Parse resume
        let resumeData;
        
        if (elements.resumeText.value.trim()) {
            try {
                resumeData = JSON.parse(elements.resumeText.value.trim());
            } catch (e) {
                alert('Invalid resume JSON format');
                hideLoading();
                return;
            }
        } else if (elements.resumeFile.files.length > 0) {
            // For simplicity, we'll assume the file is already loaded
            // In a real app, you'd read the file and parse it
            alert('File upload not implemented yet. Please paste resume JSON.');
            hideLoading();
            return;
        } else {
            alert('Please provide a resume');
            hideLoading();
            return;
        }
        
        state.resume = resumeData;
        
        // Call API
        const response = await fetch('/api/match', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                extracted_skills: state.extractedSkills,
                resume: state.resume
            })
        });
        
        if (!response.ok) {
            throw new Error(`API error: ${response.status}`);
        }
        
        // Parse response
        state.scoredResume = await response.json();
        
        // Display matching scores
        displayMatchingScores();
        
        // Enable transform button
        elements.transformBtn.disabled = false;
        
        // Navigate to next step
        navigateToStep(3);
    } catch (error) {
        console.error('Match error:', error);
        alert(`Error matching resume: ${error.message}`);
    } finally {
        hideLoading();
    }
}

// Display matching scores
function displayMatchingScores() {
    if (!state.scoredResume) return;
    
    const { professional_experience, projects } = state.scoredResume;
    
    let html = '<div class="accordion" id="matchingAccordion">';
    
    // Professional Experience
    html += `
        <div class="accordion-item">
            <h2 class="accordion-header">
                <button class="accordion-button" type="button" data-bs-toggle="collapse" data-bs-target="#experienceCollapse" aria-expanded="true" aria-controls="experienceCollapse">
                    Professional Experience
                </button>
            </h2>
            <div id="experienceCollapse" class="accordion-collapse collapse show" data-bs-parent="#matchingAccordion">
                <div class="accordion-body">
    `;
    
    professional_experience.forEach(exp => {
        const scoreClass = exp.score >= 8 ? 'score-high' : (exp.score >= 5 ? 'score-medium' : 'score-low');
        
        html += `
            <div class="entry-card">
                <div class="entry-header">
                    <h6>${exp.company} - ${exp.position} <span class="score-badge ${scoreClass}">${exp.score}</span></h6>
                    <div><small>Matching Skills: ${exp.matching_skills.join(', ')}</small></div>
                </div>
                <div class="highlights-list">
        `;
        
        exp.highlights.forEach(highlight => {
            const highlightScoreClass = highlight.score >= 8 ? 'score-high' : (highlight.score >= 5 ? 'score-medium' : 'score-low');
            
            html += `
                <div class="highlight-item">
                    <div><span class="score-badge ${highlightScoreClass}">${highlight.score}</span> ${highlight.text}</div>
                    <div><small>Matching Skills: ${highlight.matching_skills.join(', ')}</small></div>
                </div>
            `;
        });
        
        html += `
                </div>
            </div>
        `;
    });
    
    html += `
                </div>
            </div>
        </div>
    `;
    
    // Projects
    html += `
        <div class="accordion-item">
            <h2 class="accordion-header">
                <button class="accordion-button collapsed" type="button" data-bs-toggle="collapse" data-bs-target="#projectsCollapse" aria-expanded="false" aria-controls="projectsCollapse">
                    Projects
                </button>
            </h2>
            <div id="projectsCollapse" class="accordion-collapse collapse" data-bs-parent="#matchingAccordion">
                <div class="accordion-body">
    `;
    
    projects.forEach(proj => {
        const scoreClass = proj.score >= 8 ? 'score-high' : (proj.score >= 5 ? 'score-medium' : 'score-low');
        
        html += `
            <div class="entry-card">
                <div class="entry-header">
                    <h6>${proj.name} <span class="score-badge ${scoreClass}">${proj.score}</span></h6>
                    <div><small>Matching Skills: ${proj.matching_skills.join(', ')}</small></div>
                </div>
                <div class="highlights-list">
        `;
        
        proj.highlights.forEach(highlight => {
            const highlightScoreClass = highlight.score >= 8 ? 'score-high' : (highlight.score >= 5 ? 'score-medium' : 'score-low');
            
            html += `
                <div class="highlight-item">
                    <div><span class="score-badge ${highlightScoreClass}">${highlight.score}</span> ${highlight.text}</div>
                    <div><small>Matching Skills: ${highlight.matching_skills.join(', ')}</small></div>
                </div>
            `;
        });
        
        html += `
                </div>
            </div>
        `;
    });
    
    html += `
                </div>
            </div>
        </div>
    `;
    
    html += '</div>';
    
    elements.matchingScores.innerHTML = html;
}

// Handle transform button click
async function handleTransform() {
    try {
        showLoading('Transforming high-scoring resume entries...');
        
        // Call API
        const response = await fetch('/api/transform', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                scored_resume: state.scoredResume,
                extracted_skills: state.extractedSkills,
                min_score: state.minScore
            })
        });
        
        if (!response.ok) {
            throw new Error(`API error: ${response.status}`);
        }
        
        // Parse response
        state.transformedResume = await response.json();
        
        // Display transformed entries
        displayTransformedEntries();
        
        // Enable download button
        elements.downloadBtn.disabled = false;
        
        // Navigate to next step
        navigateToStep(4);
    } catch (error) {
        console.error('Transform error:', error);
        alert(`Error transforming resume: ${error.message}`);
    } finally {
        hideLoading();
    }
}

// Display transformed entries
function displayTransformedEntries() {
    if (!state.transformedResume) return;
    
    const { professional_experience, projects } = state.transformedResume;
    
    let html = '<div class="mb-4">';
    
    // Professional Experience
    html += '<h5>Professional Experience</h5>';
    
    professional_experience.forEach(exp => {
        html += `
            <div class="entry-card">
                <div class="entry-header">
                    <h6>${exp.company} - ${exp.position}</h6>
                </div>
                <div class="highlights-list">
        `;
        
        exp.highlights.forEach((highlight, index) => {
            const highlightId = `exp_${exp.company.replace(/\s+/g, '_')}_${index}`;
            
            html += `
                <div class="highlight-item mb-4">
                    <div class="row mb-2">
                        <div class="col-6">
                            <strong>Original:</strong>
                            <p>${highlight.original}</p>
                        </div>
                        <div class="col-6">
                            <strong>Transformed:</strong>
                            <p id="${highlightId}_transformed">${highlightEmphasizedSkills(highlight.transformed, highlight.emphasized_skills)}</p>
                        </div>
                    </div>
                    <div class="row">
                        <div class="col-6">
                            <label class="toggle-switch">
                                <input type="checkbox" checked id="${highlightId}_toggle">
                                <span class="toggle-slider"></span>
                            </label>
                            <span class="toggle-label">Keep this bullet point</span>
                        </div>
                        <div class="col-6">
                            <button class="btn btn-sm btn-outline-primary" 
                                    onclick="generateAlternative('${highlightId}', '${highlight.original}', ${JSON.stringify(highlight.emphasized_skills).replace(/"/g, '&quot;')})">
                                Generate Alternative
                            </button>
                        </div>
                    </div>
                </div>
            `;
        });
        
        html += `
                </div>
            </div>
        `;
    });
    
    // Projects
    html += '<h5 class="mt-4">Projects</h5>';
    
    projects.forEach(proj => {
        html += `
            <div class="entry-card">
                <div class="entry-header">
                    <h6>${proj.name}</h6>
                </div>
                <div class="highlights-list">
        `;
        
        proj.highlights.forEach((highlight, index) => {
            const highlightId = `proj_${proj.name.replace(/\s+/g, '_')}_${index}`;
            
            html += `
                <div class="highlight-item mb-4">
                    <div class="row mb-2">
                        <div class="col-6">
                            <strong>Original:</strong>
                            <p>${highlight.original}</p>
                        </div>
                        <div class="col-6">
                            <strong>Transformed:</strong>
                            <p id="${highlightId}_transformed">${highlightEmphasizedSkills(highlight.transformed, highlight.emphasized_skills)}</p>
                        </div>
                    </div>
                    <div class="row">
                        <div class="col-6">
                            <label class="toggle-switch">
                                <input type="checkbox" checked id="${highlightId}_toggle">
                                <span class="toggle-slider"></span>
                            </label>
                            <span class="toggle-label">Keep this bullet point</span>
                        </div>
                        <div class="col-6">
                            <button class="btn btn-sm btn-outline-primary" 
                                    onclick="generateAlternative('${highlightId}', '${highlight.original}', ${JSON.stringify(highlight.emphasized_skills).replace(/"/g, '&quot;')})">
                                Generate Alternative
                            </button>
                        </div>
                    </div>
                </div>
            `;
        });
        
        html += `
                </div>
            </div>
        `;
    });
    
    html += '</div>';
    
    elements.transformedEntries.innerHTML = html;
}

// Highlight emphasized skills in transformed text
function highlightEmphasizedSkills(text, skills) {
    let highlightedText = text;
    
    skills.forEach(skill => {
        // Create a case-insensitive regex to match the skill
        const regex = new RegExp(`(${skill})`, 'gi');
        highlightedText = highlightedText.replace(regex, '<span class="emphasized-skill">$1</span>');
    });
    
    return highlightedText;
}

// Generate alternative for a bullet point
async function generateAlternative(highlightId, bulletPoint, matchingSkills) {
    try {
        showLoading('Generating alternative...');
        
        // Call API
        const response = await fetch('/api/alternative', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                bullet_point: bulletPoint,
                matching_skills: matchingSkills
            })
        });
        
        if (!response.ok) {
            throw new Error(`API error: ${response.status}`);
        }
        
        // Parse response
        const data = await response.json();
        
        // Update the transformed text
        const transformedElement = document.getElementById(`${highlightId}_transformed`);
        if (transformedElement) {
            transformedElement.innerHTML = highlightEmphasizedSkills(data.alternative, matchingSkills);
        }
    } catch (error) {
        console.error('Generate alternative error:', error);
        alert(`Error generating alternative: ${error.message}`);
    } finally {
        hideLoading();
    }
}

// Handle download button click
function handleDownload() {
    if (!state.transformedResume) return;
    
    // Create a tailored resume object
    const tailoredResume = {
        ...state.resume,
    };
    
    // Replace the original highlights with the transformed ones that are toggled on
    if (tailoredResume.CV && tailoredResume.CV.sections) {
        // Professional Experience
        if (tailoredResume.CV.sections.ProfessionalExperience) {
            state.transformedResume.professional_experience.forEach(transformedExp => {
                const originalExp = tailoredResume.CV.sections.ProfessionalExperience.find(
                    exp => exp.company === transformedExp.company && exp.position === transformedExp.position
                );
                
                if (originalExp) {
                    const newHighlights = [];
                    
                    transformedExp.highlights.forEach((highlight, index) => {
                        const highlightId = `exp_${transformedExp.company.replace(/\s+/g, '_')}_${index}`;
                        const toggleElement = document.getElementById(`${highlightId}_toggle`);
                        
                        if (toggleElement && toggleElement.checked) {
                            const transformedElement = document.getElementById(`${highlightId}_transformed`);
                            if (transformedElement) {
                                // Get the text without HTML tags
                                const transformedText = transformedElement.textContent;
                                newHighlights.push(transformedText);
                            }
                        }
                    });
                    
                    originalExp.highlights = newHighlights;
                }
            });
        }
        
        // Projects
        if (tailoredResume.CV.sections.Projects) {
            state.transformedResume.projects.forEach(transformedProj => {
                const originalProj = tailoredResume.CV.sections.Projects.find(
                    proj => proj.name === transformedProj.name
                );
                
                if (originalProj) {
                    const newHighlights = [];
                    
                    transformedProj.highlights.forEach((highlight, index) => {
                        const highlightId = `proj_${transformedProj.name.replace(/\s+/g, '_')}_${index}`;
                        const toggleElement = document.getElementById(`${highlightId}_toggle`);
                        
                        if (toggleElement && toggleElement.checked) {
                            const transformedElement = document.getElementById(`${highlightId}_transformed`);
                            if (transformedElement) {
                                // Get the text without HTML tags
                                const transformedText = transformedElement.textContent;
                                newHighlights.push(transformedText);
                            }
                        }
                    });
                    
                    originalProj.highlights = newHighlights;
                }
            });
        }
    }
    
    // Create a JSON blob
    const blob = new Blob([JSON.stringify(tailoredResume, null, 2)], { type: 'application/json' });
    
    // Create a download link
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'tailored_resume.json';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

// Make generateAlternative available globally
window.generateAlternative = generateAlternative;

// Initialize the application when the DOM is loaded
document.addEventListener('DOMContentLoaded', init);
