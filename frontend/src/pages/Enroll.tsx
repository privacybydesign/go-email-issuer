import { useTranslation } from "react-i18next";
import { Link, useNavigate } from "react-router-dom";
import { useAppContext } from "../AppContext";
import i18n from "../i18n";
import { useEffect, useState } from "react";
type VerifyResponse = {
  jwt: string;
  irma_server_url: string;
  expires: number;
};

export default function EnrollPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [message, setMessage] = useState<string | undefined>(undefined);
  const [errorMessage, setErrorMessage] = useState<string | undefined>(
    undefined
  );
  const { email, setEmail } = useAppContext();
  const [token, setToken] = useState("");

  useEffect(() => {
    setMessage(t("email_sent"));
  }, [email]);

  // user clicks the link in the email to verify and start the issuance
  const VerifyAndStartIssuance = async (token: string) => {
    try {
      // send email and token to verify endpoint to see if the token is valid for this email
      const response = await fetch("/verify", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          token: token,
        }),
      });

      if (response.ok) {
        // Start enrollment process
        const res: VerifyResponse = await response.json();
        const expiry = new Date(res.expires * 1000);
        setMessage(
          `The verification link expires in ${Math.floor(
            (expiry.getTime() - Date.now()) / 1000 / 60
          )} minutes`
        );

        import("@privacybydesign/yivi-frontend").then((yivi) => {
          const issuance = yivi.newPopup({
            language: i18n.language,
            session: {
              url: res.irma_server_url,
              start: {
                method: "POST",
                headers: {
                  "Content-Type": "text/plain",
                },
                body: res.jwt,
              },
              result: false,
            },
          });
          issuance
            .start()
            .then(() => {
              setMessage(t("email_add_success"));
              setEmail("");
              setToken("");
              navigate(`/${i18n.language}/done`);
            })
            .catch((e: string) => {
              if (e === "Aborted") {
                setErrorMessage(t("email_add_cancel"));
              } else {
                setErrorMessage(t("email_add_error"));
              }
            });
        });
        return;
      }
    } catch (error) {
      navigate(`/${i18n.language}/error`);
    }
  };

  useEffect(() => {
    const hash = window.location.hash;
    const token = hash.replace("#verify:", "");

    if (hash) {
      const match = hash.match(/^#verify:([^:]+@[^\s:]+):(.+)$/);
      if (!match) {
        navigate(`/${i18n.language}/error`);
        return;
      }

      const tokenTime = match[2] ? parseInt(match[2]) : 0;
      if (tokenTime < Date.now() - 5 * 60 * 1000) {
        setErrorMessage(t("error_link_expired"));
      }

      VerifyAndStartIssuance(token);
    }
  }, [navigate, t]);

  return (
    <>
      <div id="container">
        <header>
          <h1>{t("index_header")}</h1>
        </header>
        <main>
          <div className="sms-form">
            <div id="block-token">
              {!errorMessage && message && (
                <div
                  id="status-bar"
                  className="alert alert-success"
                  role="alert"
                >
                  <div className="status-container">
                    <div id="status">{message}</div>
                  </div>
                </div>
              )}
              {errorMessage && (
                <div
                  id="status-bar"
                  className="alert alert-danger"
                  role="alert"
                >
                  <div className="status-container">
                    <div id="status">{errorMessage}</div>
                  </div>
                </div>
              )}
              <p>{t("receive_email")}</p>
              <b>{t("steps")}</b>
              <ol>
                <li>{t("step_1")}</li>
                <li>{t("step_2")}</li>
                <li>{t("step_3")}</li>
                <li>{t("step_4")}</li>
                <li>{t("step_5")}</li>
              </ol>
            </div>
          </div>
        </main>
        <footer>
          <div className="actions">
            <Link to={`/${i18n.language}/validate`} id="back-button">
              {t("back")}
            </Link>
          </div>
        </footer>
      </div>
    </>
  );
}
