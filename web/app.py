import streamlit as st
import requests
import json
from typing import Dict, Any

st.set_page_config(page_title="Resume Optimizer", layout="wide")

st.title("Resume Optimizer")
st.markdown(
    """
Make your resume look like it was made for this specific job!
"""
)

col1, col2 = st.columns(2)

with col1:
    st.subheader("Job Description")
    jd_text = st.text_area("Paste the job description here", height=400)

with col2:
    st.subheader("Your Resume")
    resume_text = st.text_area("Paste your resume here", height=400)

if st.button("Optimize Resume"):
    if not jd_text or not resume_text:
        st.error("Please provide both job description and resume text.")
    else:
        with st.spinner("Optimizing your resume..."):
            try:
                payload = {"jobDescText": jd_text, "resume": resume_text}

                response = requests.post(
                    "http://localhost:8080/api/optimize",
                    json=payload,
                    headers={"Content-Type": "application/json"},
                )

                if response.status_code == 200:
                    result = response.json()

                    st.subheader("Extracted Skills")
                    extracted_skills = result.get("extractedSkills", {})

                    company_info = extracted_skills.get("company_info", {})
                    st.write(f"**Company:** {company_info.get('name', 'N/A')}")
                    st.write(f"**Position:** {company_info.get('position', 'N/A')}")
                    st.write(f"**Level:** {company_info.get('level', 'N/A')}")

                    col1, col2 = st.columns(2)
                    with col1:
                        st.write("Required Skills:")
                        for skill in extracted_skills.get("required_skills", []):
                            st.write(f"- {skill['name']}")
                            if skill.get("context"):
                                st.write(f"  *Context: {skill['context']}*")

                    with col2:
                        st.write("Nice to Have Skills:")
                        for skill in extracted_skills.get("nice_to_have_skills", []):
                            st.write(f"- {skill['name']}")
                            if skill.get("context"):
                                st.write(f"  *Context: {skill['context']}*")

                    st.subheader("Resume Scoring")
                    scored_resume = result.get("scoredResume", {})

                    st.metric(
                        "Overall Resume Score",
                        f"{scored_resume.get('overall_score', 0):.1f}/10",
                    )

                    st.write("Experience Scores:")
                    for exp in scored_resume.get("professional_experience", []):
                        with st.expander(
                            f"{exp.get('company', '')} - {exp.get('position', '')}: {exp.get('score', 0):.1f}/10"
                        ):
                            st.write("Matching Skills:")
                            for skill in exp.get("matching_skills", []):
                                st.write(f"- {skill}")
                            st.write("Highlights:")
                            for highlight in exp.get("highlights", []):
                                st.write(
                                    f"- {highlight.get('text', '')} (Score: {highlight.get('score', 0):.1f})"
                                )
                                if highlight.get("reasoning"):
                                    st.write(
                                        f"  *Reasoning: {highlight.get('reasoning')}*"
                                    )

                    st.write("Project Scores:")
                    projects = scored_resume.get("projects", [])
                    if not projects:
                        st.write("No projects found")
                    else:
                        for proj in projects:
                            title = proj.get("name", "Unnamed Project")
                            with st.expander(f"- {title}: {proj.get('score', 0):.1f}/10"):
                                st.write("Matching Skills:")
                                for skill in proj.get("matching_skills", []):
                                    st.write(f"- {skill}")
                                st.write("Highlights:")
                                for highlight in proj.get("highlights", []):
                                    st.write(
                                        f"- {highlight.get('text', '')} (Score: {highlight.get('score', 0):.1f})"
                                    )
                                    if highlight.get("reasoning"):
                                        st.write(
                                            f"  *Reasoning: {highlight.get('reasoning')}*"
                                        )

                    st.subheader("Suggested Optimizations Bullet Points")
                    transformed_items = result.get("transformItems", [])

                    for item in transformed_items:
                        title = (
                            f"{item.get('company', '')} - {item.get('position', '')}"
                        )
                        if not title.strip():
                            title = item.get("name", "Project")

                        score_diff = item.get("new_score", 0) - item.get(
                            "original_score", 0
                        )
                        score_text = f"{item.get('id', 'N/A')} ({item.get('original_score', 0):.1f} ↗️ +{score_diff:.1f} = {item.get('new_score', 0):.1f})"

                        with st.expander(f"{title} {score_text}"):
                            col1, col2 = st.columns(2)
                            with col1:
                                st.write("**Original:**")
                                st.write(item.get("original_text", ""))
                                if item.get("original_skills"):
                                    st.write("**Original Skills:**")
                                    for skill in item.get("original_skills", []):
                                        st.write(f"- {skill}")
                            with col2:
                                st.write("**Optimized:**")
                                st.write(item.get("transformed_text", ""))
                                if item.get("added_skills"):
                                    st.write("**Added Skills:**")
                                    for skill in item.get("added_skills", []):
                                        st.write(f"- {skill} ✨")

                            if item.get("reasoning"):
                                st.write("**Reasoning:**")
                                st.write(item.get("reasoning", ""))
                            if item.get("improvement_explanation"):
                                st.write("**Improvement Explanation:**")
                                st.write(item.get("improvement_explanation", ""))

                            # Show character counts if available
                            if item.get("char_count_original") or item.get(
                                "char_count_new"
                            ):
                                st.write("**Character Counts:**")
                                st.write(
                                    f"- Original: {item.get('char_count_original', 0)}"
                                )
                                st.write(f"- New: {item.get('char_count_new', 0)}")
                else:
                    st.error(f"Error: {response.status_code} - {response.text}")

            except Exception as e:
                st.error(f"An error occurred: {str(e)}")

