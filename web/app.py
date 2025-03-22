import streamlit as st
import requests
import os
from dotenv import load_dotenv

load_dotenv()
BACKEND_URL = os.getenv("BACKEND_URL", "http://localhost:8080")

if "results" not in st.session_state:
    st.session_state.results = None
if "jd_text" not in st.session_state:
    st.session_state.jd_text = ""
if "resume_text" not in st.session_state:
    st.session_state.resume_text = ""
if "transformed_sections" not in st.session_state:
    st.session_state.transformed_sections = {}


def display_skills(skills, title):
    st.write(f"{title}:")
    for skill in skills:
        imp = skill["importance"]
        color = "red" if imp < 3 else "green" if imp > 7 else "orange"
        text = f"**{skill['name']}** <span style='color:{color}'>●&nbsp;{imp}/10</span>"
        if skill.get("context"):
            text += f" - *{skill['context']}*"
        st.markdown(text, unsafe_allow_html=True)


def transform_section(section_data, section_idx):
    section_key = f"{section_data.get('name', '')}_{section_idx}"

    try:
        response = requests.post(
            url=f"{BACKEND_URL}/api/transformSection",
            json=section_data,
            headers={"Content-Type": "application/json"},
        )

        if response.status_code == 200:
            st.session_state.transformed_sections[section_key] = response.json()
        else:
            st.error(f"API Error: {response.status_code}")
    except Exception as e:
        st.error(f"Error: {str(e)}")


def score_resume():
    try:
        with st.spinner("Scoring your resume..."):
            response = requests.post(
                f"{BACKEND_URL}/api/score",
                json={
                    "jobDescText": st.session_state.jd_text,
                    "resume": st.session_state.resume_text,
                },
            )
            response.raise_for_status()
            st.session_state.results = response.json()
    except Exception as e:
        st.error(f"Error: {str(e)}")


def save_inputs():
    if st.session_state.get("jd_input"):
        st.session_state.jd_text = st.session_state.jd_input
    if st.session_state.get("resume_input"):
        st.session_state.resume_text = st.session_state.resume_input

    if "jd_file" in st.session_state and st.session_state.jd_file is not None:
        jd_file = st.session_state.jd_file
        if jd_file is not None:
            st.session_state.jd_text = jd_file.getvalue().decode("utf-8")

    if "resume_file" in st.session_state and st.session_state.resume_file is not None:
        resume_file = st.session_state.resume_file
        if resume_file is not None:
            st.session_state.resume_text = resume_file.getvalue().decode("utf-8")


st.set_page_config(page_title="TailorMyResume", layout="wide")
st.title("TailormyResume")
st.markdown("Make your resume look like it was made for this specific job!")

col1, col2 = st.columns(2)
with col1:
    st.subheader("Job Description")

    jd_method = st.radio(
        "Input method for job description:",
        ["Paste text", "Upload file"],
        key="jd_method",
    )

    if jd_method == "Paste text":
        st.text_area(
            "Paste job description here:",
            height=300,
            key="jd_input",
            value=st.session_state.jd_text,
        )
    else:
        st.file_uploader(
            "Upload job description file (txt, pdf, docx):",
            type=["txt", "pdf", "docx"],
            key="jd_file",
            help="Upload a text file containing the job description",
        )
        if "jd_file" in st.session_state and st.session_state.jd_file is not None:
            st.success(f"Uploaded: {st.session_state.jd_file.name}")


with col2:
    st.subheader("Your Resume")

    resume_method = st.radio(
        "Input method for resume:", ["Paste text", "Upload file"], key="resume_method"
    )

    if resume_method == "Paste text":
        st.text_area(
            "Paste resume here:",
            height=300,
            key="resume_input",
            value=st.session_state.resume_text,
        )
    else:
        st.file_uploader(
            "Upload resume file (txt, pdf, docx):",
            type=["txt", "pdf", "docx"],
            key="resume_file",
            help="Upload a text file containing your resume",
        )
        if (
            "resume_file" in st.session_state
            and st.session_state.resume_file is not None
        ):
            st.success(f"Uploaded: {st.session_state.resume_file.name}")

if st.button("Score Resume"):
    save_inputs()
    score_resume()

if st.session_state.results:
    result = st.session_state.results

    st.header("Results")
    st.metric("Overall Score", f"{result.get('overall_score', 0):.1f}/10")
    st.write(f"**Overall Comments:** {result.get('overall_comments', '')}")

    for idx, section in enumerate(result.get("sections", [])):
        section_key = f"{section.get('name', '')}_{idx}"

        with st.expander(
            f"{section.get('name', '')}: {section.get('score', 0):.1f}/10",
            expanded=False,
        ):
            st.write(section.get("score_reasoning", ""))

            col1, col2 = st.columns(2)
            with col1:
                st.subheader("Original Content")
                st.write(section.get("original_content", ""))

                if section.get("missing_skills"):
                    st.subheader("Missing Skills")
                    for skill in section.get("missing_skills", []):
                        st.write(f"- {skill['name']} ({skill['importance']}/10)")

            with col2:
                if section_key in st.session_state.transformed_sections:
                    transformed = st.session_state.transformed_sections[section_key]
                    st.subheader("Transformed Content")
                    st.write(transformed.get("improvement_explanation", ""))

                    for item in transformed.get("items", []):
                        st.markdown(f"• {item.get('transformed_bullet', '')}")
                else:
                    if st.button("🪄 Transform", key=f"transform_{section_key}"):
                        transform_section(section, idx)
                        st.rerun()
