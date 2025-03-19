import streamlit as st
import requests
import json
import re
from typing import Dict, Any

st.set_page_config(
    page_title="Resume Tailor",
    page_icon="📝",
    layout="wide",
    initial_sidebar_state="collapsed",
)

st.markdown(
    """
<style>
    .highlight {
        background-color: #f0f8ff;
        padding: 5px;
        border-radius: 5px;
        font-weight: bold;
        color: #0066cc;
    }
    .skill-tag {
        display: inline-block;
        padding: 3px 8px;
        margin: 2px;
        border-radius: 10px;
        font-size: 0.8em;
    }
    .required {
        background-color: #91220a ;
        border: 1px solid #9ec5fe;
    }
    .nice-to-have {
        background-color: #005d10;
        border: 1px solid #a3cfbb;
    }
    .score-high {
        color: white;
        background-color: #198754;
        padding: 2px 6px;
        border-radius: 10px;
    }
    .score-medium {
        color: white;
        background-color: #fd7e14;
        padding: 2px 6px;
        border-radius: 10px;
    }
    .score-low {
        color: white;
        background-color: #dc3545;
        padding: 2px 6px;
        border-radius: 10px;
    }
</style>
""",
    unsafe_allow_html=True,
)

st.title("Resume Tailor 📝")
st.markdown("**Make your resume scream 'I'm perfect for this job!' without lying.**")

if "extracted_skills" not in st.session_state:
    st.session_state.extracted_skills = None
if "scored_resume" not in st.session_state:
    st.session_state.scored_resume = None
if "transformed_resume" not in st.session_state:
    st.session_state.transformed_resume = None

BACKEND_URL = "http://localhost:8080"


def call_api(endpoint, data, is_form=False):
    """
    Call the API with the given endpoint and data.

    Args:
        endpoint (str): The API endpoint to call.
        data (dict): The data to send to the API.
        is_form (bool, optional): Whether to send as form data. Defaults to False.

    Returns:
        The parsed JSON response or None if there was an error.
    """
    api_url = f"{BACKEND_URL}/api/{endpoint}"

    try:
        if is_form:
            response = requests.post(api_url, data=data)
        else:
            headers = {"Content-Type": "application/json"}
            response = requests.post(api_url, json=data, headers=headers)

        if response.status_code != 200:
            st.error(f"API Error: {response.status_code} - {response.text}")
            return None

        return response.json()

    except Exception as e:
        st.error(f"Error calling API: {str(e)}")
        return None


def format_score(score: int) -> str:
    if score >= 8:
        return f"<span class='score-high'>{score}</span>"
    elif score >= 5:
        return f"<span class='score-medium'>{score}</span>"
    else:
        return f"<span class='score-low'>{score}</span>"


def highlight_skills(text: str, skills: list) -> str:
    if not skills:
        return text

    result = text
    for skill in skills:
        pattern = re.compile(re.escape(skill), re.IGNORECASE)
        result = pattern.sub(f"<span class='highlight'>{skill}</span>", result)

    return result


tab1, tab2, tab3 = st.tabs(["Job Analysis", "Resume Matching", "Resume Transformation"])

with tab1:
    st.header("Step 1: Analyze Job Description")

    job_desc = st.text_area(
        "Paste the job description here",
        height=300,
        placeholder="Copy and paste the full job description from the job posting...",
        key="job_desc",
    )

    extract_btn = st.button("Extract Skills", type="primary")

    if extract_btn:
        if not job_desc.strip():
            st.error("Please enter a job description")
        else:
            extracted_data = call_api(
                "extract", {"jobDescText": job_desc}, is_form=True
            )

            if extracted_data:
                st.session_state.extracted_skills = extracted_data
                st.success("Skills extracted successfully!")

    if st.session_state.extracted_skills:
        st.subheader("Extracted Skills")

        skills = st.session_state.extracted_skills
        company_info = skills.get("company_info", {})

        col1, col2, col3 = st.columns(3)
        with col1:
            st.info(f"**Company:** {company_info.get('name', 'N/A')}")
        with col2:
            st.info(f"**Position:** {company_info.get('position', 'N/A')}")
        with col3:
            st.info(f"**Level:** {company_info.get('level', 'N/A')}")

        st.markdown("### Required Skills")
        required_skills_html = ""
        for skill in skills.get("required_skills", []):
            required_skills_html += f"<span class='skill-tag required'>{skill['name']}: '{skill['context']}'</span> "
        st.markdown(required_skills_html, unsafe_allow_html=True)

        st.markdown("### Nice-to-Have Skills")
        nice_skills_html = ""
        for skill in skills.get("nice_to_have_skills", []):
            nice_skills_html += f"<span class='skill-tag nice-to-have'>{skill['name']}: '{skill['context']}'</span> "
        st.markdown(nice_skills_html, unsafe_allow_html=True)


# For tab2 - Resume Matching
with tab2:
    st.header("Step 2: Match Your Resume")

    # Check if user has completed step 1
    if not st.session_state.extracted_skills:
        st.warning("Please complete Step 1 (Job Analysis) first")
    else:
        st.subheader("Enter Your Resume")

        # Text area for resume input
        resume_text = st.text_area(
            "Paste your resume text here",
            height=300,
            placeholder="Copy and paste your full resume text...",
            key="resume_text",
        )

        match_btn = st.button("Match Resume", type="primary")

        if match_btn:
            if not resume_text.strip():
                st.error("Please enter your resume text")
            else:
                with st.spinner("Analyzing resume match..."):
                    matched_data = call_api(
                        "match",
                        {
                            "extractedSkills": json.dumps(
                                st.session_state.extracted_skills
                            ),
                            "resumeText": resume_text,
                        },
                        is_form=True,
                    )

                    if matched_data:
                        st.session_state.matched_resume = matched_data
                        st.success("Resume matched successfully!")

        if "matched_resume" in st.session_state and st.session_state.matched_resume:
            st.subheader("Match Results")

            if "overall_score" in st.session_state.matched_resume:
                score = st.session_state.matched_resume["overall_score"]
                st.markdown(f"### Overall Match Score: {score}/10")
                if score >= 5:
                    if st.button("Proceed to Resume Transformation", type="primary"):
                        st.session_state.active_tab = 2  # Move to the third tab
                else:
                    st.warning(
                        "Your resume has a low match score. Consider adding more relevant experience before proceeding."
                    )
            if "professional_experience" in st.session_state.matched_resume:
                st.markdown("### Professional Experience")

                for exp in st.session_state.matched_resume["professional_experience"]:
                    with st.expander(
                        f"{exp['company']} - {exp['position']} (Match Score: {exp['score']}/10)",
                        expanded=True,
                    ):
                        st.markdown("**Matching Skills:**")
                        skills_html = ""
                        for skill in exp.get("matching_skills", []):
                            skills_html += (
                                f"<span class='skill-tag matched'>{skill}</span> "
                            )
                        st.markdown(skills_html, unsafe_allow_html=True)

                        st.markdown("**Highlights:**")
                        for highlight in exp.get("highlights", []):
                            col1, col2 = st.columns([4, 1])
                            with col1:
                                st.markdown(highlight["text"])
                            with col2:
                                score_color = (
                                    "green"
                                    if highlight["score"] >= 7
                                    else "orange" if highlight["score"] >= 5 else "red"
                                )
                                st.markdown(
                                    f"<span style='color:{score_color};font-weight:bold;'>{highlight['score']}/10</span>",
                                    unsafe_allow_html=True,
                                )

                            if highlight.get("matching_skills"):
                                mini_skills_html = ""
                                for skill in highlight.get("matching_skills", []):
                                    mini_skills_html += f"<span class='skill-tag mini-matched'>{skill}</span> "
                                st.markdown(mini_skills_html, unsafe_allow_html=True)
                            st.divider()

            if "projects" in st.session_state.matched_resume:
                st.markdown("### Projects")

                for project in st.session_state.matched_resume["projects"]:
                    with st.expander(
                        f"{project['name']} (Match Score: {project['score']}/10)",
                        expanded=True,
                    ):
                        st.markdown("**Matching Skills:**")
                        skills_html = ""
                        for skill in project.get("matching_skills", []):
                            skills_html += (
                                f"<span class='skill-tag matched'>{skill}</span> "
                            )
                        st.markdown(skills_html, unsafe_allow_html=True)

                        st.markdown("**Highlights:**")
                        for highlight in project.get("highlights", []):
                            col1, col2 = st.columns([4, 1])
                            with col1:
                                st.markdown(highlight["text"])
                            with col2:
                                score_color = (
                                    "green"
                                    if highlight["score"] >= 7
                                    else "orange" if highlight["score"] >= 5 else "red"
                                )
                                st.markdown(
                                    f"<span style='color:{score_color};font-weight:bold;'>{highlight['score']}/10</span>",
                                    unsafe_allow_html=True,
                                )

                            if highlight.get("matching_skills"):
                                mini_skills_html = ""
                                for skill in highlight.get("matching_skills", []):
                                    mini_skills_html += f"<span class='skill-tag mini-matched'>{skill}</span> "
                                st.markdown(mini_skills_html, unsafe_allow_html=True)
                            st.divider()


with tab3:
    st.header("Step 3: Transform Your Resume")

    if not st.session_state.get("extracted_skills"):
        st.warning("Please complete Step 1 (Job Analysis) first")
    elif not st.session_state.get("matched_resume"):
        st.warning("Please complete Step 2 (Resume Matching) first")
    else:
        st.subheader("Optimize Your Resume")

        st.markdown(
            """
        This step will transform your top-scoring resume bullets to better match the job requirements.
        We'll focus on bullets with scores of 7 or higher, as these are your strongest matches.
        """
        )

        st.markdown("### Transformation Settings")
        col1, col2 = st.columns(2)
        with col1:
            min_score = st.slider("Minimum score to transform", 1, 10, 7)
        with col2:
            emphasis_level = st.select_slider(
                "Keyword emphasis level",
                options=["Subtle", "Moderate", "Strong"],
                value="Moderate",
            )

        transform_btn = st.button("Transform Resume", type="primary")

        if transform_btn:
            with st.spinner("Optimizing your resume..."):
                high_scoring_items = []

                for exp in st.session_state.matched_resume.get(
                    "professional_experience", []
                ):
                    exp_items = []
                    for highlight in exp.get("highlights", []):
                        if highlight.get("score", 0) >= min_score:
                            exp_items.append(
                                {
                                    "original_text": highlight["text"],
                                    "matching_skills": highlight.get(
                                        "matching_skills", []
                                    ),
                                    "section": "experience",
                                    "company": exp.get("company", ""),
                                    "position": exp.get("position", ""),
                                }
                            )
                    high_scoring_items.extend(exp_items)

                for project in st.session_state.matched_resume.get("projects", []):
                    project_items = []
                    for highlight in project.get("highlights", []):
                        if highlight.get("score", 0) >= min_score:
                            project_items.append(
                                {
                                    "original_text": highlight["text"],
                                    "matching_skills": highlight.get(
                                        "matching_skills", []
                                    ),
                                    "section": "project",
                                    "name": project.get("name", ""),
                                }
                            )
                    high_scoring_items.extend(project_items)

                if not high_scoring_items:
                    st.warning(
                        f"No bullet points with score {min_score} or higher found. Try lowering the minimum score."
                    )
                else:
                    # Call the transform API
                    transformed_data = call_api(
                        "transform",
                        {
                            "extractedSkills": json.dumps(
                                st.session_state.extracted_skills
                            ),
                            "items": json.dumps(high_scoring_items),
                            "emphasisLevel": emphasis_level,
                        },
                    )

                    if transformed_data:
                        st.session_state.transformed_resume = transformed_data
                        st.success(
                            f"Successfully transformed {len(transformed_data.get('items', []))} bullet points!"
                        )

        if (
            "transformed_resume" in st.session_state
            and st.session_state.transformed_resume
        ):
            st.subheader("Transformed Resume")

            transformed_items = st.session_state.transformed_resume.get("items", [])

            experience_items = {}
            project_items = {}

            for item in transformed_items:
                if item.get("section") == "experience":
                    company_key = f"{item.get('company')} - {item.get('position')}"
                    if company_key not in experience_items:
                        experience_items[company_key] = []
                    experience_items[company_key].append(item)
                elif item.get("section") == "project":
                    project_name = item.get("name", "")
                    if project_name not in project_items:
                        project_items[project_name] = []
                    project_items[project_name].append(item)

            if experience_items:
                st.markdown("### Professional Experience")

                for company, items in experience_items.items():
                    with st.expander(company, expanded=True):
                        for item in items:
                            col1, col2 = st.columns(2)

                            with col1:
                                st.markdown("**Original:**")
                                st.markdown(item.get("original_text", ""))

                            with col2:
                                st.markdown("**Optimized:**")
                                optimized_text = item.get("transformed_text", "")
                                # Highlight keywords
                                for skill in item.get("matching_skills", []):
                                    # Case-insensitive replace with bold
                                    pattern = re.compile(
                                        re.escape(skill), re.IGNORECASE
                                    )
                                    optimized_text = pattern.sub(
                                        f"**{skill}**", optimized_text
                                    )
                                st.markdown(optimized_text)

                            # Actions for this bullet
                            col1, col2, col3 = st.columns([1, 1, 2])
                            with col1:
                                if st.button(
                                    f"Keep Original #{item.get('id', '')}",
                                    key=f"keep_orig_{item.get('id', '')}",
                                ):
                                    # Logic to keep original
                                    st.session_state.final_choices = (
                                        st.session_state.get("final_choices", {})
                                    )
                                    st.session_state.final_choices[
                                        item.get("id", "")
                                    ] = {
                                        "text": item.get("original_text", ""),
                                        "is_original": True,
                                    }
                                    st.success("Original version selected!")

                            with col2:
                                if st.button(
                                    f"Keep Optimized #{item.get('id', '')}",
                                    key=f"keep_opt_{item.get('id', '')}",
                                ):
                                    # Logic to keep optimized
                                    st.session_state.final_choices = (
                                        st.session_state.get("final_choices", {})
                                    )
                                    st.session_state.final_choices[
                                        item.get("id", "")
                                    ] = {
                                        "text": item.get("transformed_text", ""),
                                        "is_original": False,
                                    }
                                    st.success("Optimized version selected!")

            if project_items:
                st.markdown("### Projects")

                for project_name, items in project_items.items():
                    with st.expander(project_name, expanded=True):
                        for item in items:
                            col1, col2 = st.columns(2)

                            with col1:
                                st.markdown("**Original:**")
                                st.markdown(item.get("original_text", ""))

                            with col2:
                                st.markdown("**Optimized:**")
                                optimized_text = item.get("transformed_text", "")
                                # Highlight keywords
                                for skill in item.get("matching_skills", []):
                                    pattern = re.compile(
                                        re.escape(skill), re.IGNORECASE
                                    )
                                    optimized_text = pattern.sub(
                                        f"**{skill}**", optimized_text
                                    )
                                st.markdown(optimized_text)

                            col1, col2, col3 = st.columns([1, 1, 2])
                            with col1:
                                if st.button(
                                    f"Keep Original #{item.get('id', '')}",
                                    key=f"p_keep_orig_{item.get('id', '')}",
                                ):
                                    # Logic to keep original
                                    st.session_state.final_choices = (
                                        st.session_state.get("final_choices", {})
                                    )
                                    st.session_state.final_choices[
                                        item.get("id", "")
                                    ] = {
                                        "text": item.get("original_text", ""),
                                        "is_original": True,
                                    }
                                    st.success("Original version selected!")

                            with col2:
                                if st.button(
                                    f"Keep Optimized #{item.get('id', '')}",
                                    key=f"p_keep_opt_{item.get('id', '')}",
                                ):
                                    # Logic to keep optimized
                                    st.session_state.final_choices = (
                                        st.session_state.get("final_choices", {})
                                    )
                                    st.session_state.final_choices[
                                        item.get("id", "")
                                    ] = {
                                        "text": item.get("transformed_text", ""),
                                        "is_original": False,
                                    }
                                    st.success("Optimized version selected!")

                            with col3:
                                if st.button(
                                    f"Generate Alternative #{item.get('id', '')}",
                                    key=f"p_alt_{item.get('id', '')}",
                                ):
                                    # Call API to generate alternative
                                    alternative_data = call_api(
                                        "alternative",
                                        {
                                            "extractedSkills": json.dumps(
                                                st.session_state.extracted_skills
                                            ),
                                            "originalText": item.get(
                                                "original_text", ""
                                            ),
                                            "matchingSkills": json.dumps(
                                                item.get("matching_skills", [])
                                            ),
                                            "emphasisLevel": emphasis_level,
                                        },
                                    )

                                    if (
                                        alternative_data
                                        and "alternative_text" in alternative_data
                                    ):
                                        # Update the item with the alternative
                                        for i, t_item in enumerate(transformed_items):
                                            if t_item.get("id") == item.get("id"):
                                                transformed_items[i][
                                                    "alternative_text"
                                                ] = alternative_data["alternative_text"]
                                                break
                                        st.session_state.transformed_resume["items"] = (
                                            transformed_items
                                        )
                                        st.success("Alternative generated!")
                                        st.experimental_rerun()

                            # Show alternative if available
                            if "alternative_text" in item:
                                st.markdown("**Alternative:**")
                                alt_text = item.get("alternative_text", "")
                                # Highlight keywords
                                for skill in item.get("matching_skills", []):
                                    pattern = re.compile(
                                        re.escape(skill), re.IGNORECASE
                                    )
                                    alt_text = pattern.sub(f"**{skill}**", alt_text)
                                st.markdown(alt_text)

                                if st.button(
                                    f"Keep Alternative #{item.get('id', '')}",
                                    key=f"p_keep_alt_{item.get('id', '')}",
                                ):
                                    # Logic to keep alternative
                                    st.session_state.final_choices = (
                                        st.session_state.get("final_choices", {})
                                    )
                                    st.session_state.final_choices[
                                        item.get("id", "")
                                    ] = {
                                        "text": item.get("alternative_text", ""),
                                        "is_original": False,
                                    }
                                    st.success("Alternative version selected!")

                            st.divider()

            # Export button and final resume view
            if st.session_state.get("final_choices"):
                st.subheader("Your Optimized Resume")

                # Count selections
                original_count = sum(
                    1
                    for item in st.session_state.final_choices.values()
                    if item.get("is_original", False)
                )
                optimized_count = len(st.session_state.final_choices) - original_count

                st.info(
                    f"You've selected {len(st.session_state.final_choices)} bullet points: {original_count} original and {optimized_count} optimized."
                )

                # Show the selected bullets
                selected_bullets = ""
                for item_id, choice in st.session_state.final_choices.items():
                    selected_bullets += f"• {choice['text']}\n\n"

                st.text_area(
                    "Your Selected Bullet Points", selected_bullets, height=300
                )

                # Export options
                export_col1, export_col2 = st.columns(2)
                with export_col1:
                    if st.button("Copy to Clipboard", type="primary"):
                        # Use JavaScript to copy to clipboard
                        st.write(
                            """
                        <script>
                        navigator.clipboard.writeText(`%s`).then(function() {
                            alert('Copied to clipboard!');
                        }, function() {
                            alert('Failed to copy!');
                        });
                        </script>
                        """
                            % selected_bullets,
                            unsafe_allow_html=True,
                        )
                        st.success("Copied to clipboard!")

                with export_col2:
                    if st.download_button(
                        label="Download as Text",
                        data=selected_bullets,
                        file_name="optimized_resume_bullets.txt",
                        mime="text/plain",
                    ):
                        st.success("Downloaded!")
