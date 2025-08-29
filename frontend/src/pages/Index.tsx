import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useAppContext } from "../AppContext";

type ParsedEmail = { isValid: () => boolean };
const parseEmail = (input: string): ParsedEmail => ({
  isValid: () => /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(input),
});

const isEmailValid = (input: string) => {
  try {
    const email = parseEmail(input);
    return email?.isValid();
  } catch (error) {
    return false;
  }
};

export default function IndexPage() {
  const { t, i18n } = useTranslation();
  const { email, setEmail } = useAppContext();
  const isValid = isEmailValid(email || "");
  const [showError, setShowError] = useState(false);
  const navigate = useNavigate();

  const onChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const value = e.target.value;
    setEmail(value);
    if (showError && isEmailValid(value)) {
      setShowError(false);
    }
  };

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isValid) {
      setShowError(true);
      return;
    }
    navigate(`/${i18n.language}/validate`);
  };

  return (
    <>
      <form id="container" onSubmit={submit}>
        <header>
          <h1>{t("index_header")}</h1>
        </header>
        <main>
          <div className="email-form">
            <p>{t("index_explanation")}</p>
            <p>{t("index_multiple_numbers")}</p>
            <label htmlFor="email-input">{t("email_address")}</label>
            <input
              id="email-input"
              value={email}
              onChange={onChange}
              autoFocus
            />
            <p>
              {showError && (
                <div className="warning">{t("index_email_not_valid")}</div>
              )}
            </p>
          </div>
        </main>
        <footer>
          <div className="actions">
            <div></div>
            <button id="submit-button" type="submit">
              {t("index_start")}
            </button>
          </div>
        </footer>
      </form>
    </>
  );
}
