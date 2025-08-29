import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router-dom";
import { useAppContext } from "../AppContext";
import { useState } from "react";
import { FaCheck } from "react-icons/fa";

export default function ValidatePage() {
  const navigate = useNavigate();
  const [errorMessage, setErrorMessage] = useState<string | undefined>(
    undefined
  );
  const { t, i18n } = useTranslation();
  const { email } = useAppContext();

  const enroll = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    if (!email) {
      navigate(`/${i18n.language}/error`);
      return;
    }

    const response = await fetch("/api/send", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        email: email,
        language: i18n.language,
      }),
    });

    if (response.ok) {
      navigate(`/${i18n.language}/enroll`, {
        state: { from: "validate", message: "email_sent" },
      });
    } else {
      const data = await response.json();
      let errorCode = data.error;
      if (errorCode) {
        setErrorMessage(t(errorCode));
      } else {
        navigate(`/${i18n.language}/error`);
      }
    }
  };

  return (
    <>
      <form id="container" onSubmit={enroll}>
        <header>
          <h1>{t("validate_header")}</h1>
        </header>
        <main>
          <div className="email-form">
            {errorMessage && (
              <div id="status-bar" className="alert alert-danger" role="alert">
                <div className="status-container">
                  <div id="status">{errorMessage}</div>
                </div>
              </div>
            )}
            <p>{t("validate_explanation")}</p>
            <div
              style={{
                position: "relative",
                display: "inline-block",
                width: "100%",
              }}
            >
              <input type="email" value={email} disabled />
              <FaCheck
                color="green"
                size={18}
                style={{
                  position: "absolute",
                  right: "0.75rem",
                  top: "50%",
                  transform: "translateY(-50%)",
                  pointerEvents: "none",
                }}
              />
            </div>{" "}
          </div>
        </main>
        <footer>
          <div className="actions">
            <Link to={`/${i18n.language}`} id="back-button">
              {t("back")}
            </Link>
            <button id="submit-button">{t("confirm")}</button>
          </div>
        </footer>
      </form>
    </>
  );
}
