import { useTranslation } from "react-i18next";
import { Link, useNavigate, useLocation } from "react-router-dom";
import i18n from "../i18n";
import { useEffect, useState } from "react";
type VerifyResponse = {
  jwt: string;
  irma_server_url: string;
};

export default function EnrollPage() {
  const navigate = useNavigate();
  const { t } = useTranslation();
  const [message, setMessage] = useState<string | undefined>(undefined);
  const [errorMessage, setErrorMessage] = useState<string | undefined>(
    undefined
  );
  const location = useLocation();
  const hash = location.hash;

  useEffect(() => {
    // only show the message if the user came from the validate page
    if (location.state?.from === "validate" && location.state?.message) {
      setMessage(t(location.state.message));

      window.history.replaceState({}, document.title);
    }
  }, [location.state]);
  // user clicks the link in the email to verify and start the issuance
  const VerifyAndStartIssuance = async (token: string) => {
    try {
      // send email and token to verify endpoint to see if the token is valid for this email
      const response = await fetch("/api/verify", {
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

        import("@privacybydesign/yivi-frontend").then((yivi) => {
          const issuance = yivi.newWeb({
            element: "#yivi-web-form",
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
    const token = hash.replace("#verify:", "");

    if (hash) {
      const match = hash.match(/^#verify:([^:]+@[^\s:]+):(.+)$/);
      if (!match) {
        navigate(`/${i18n.language}/error`);
        return;
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
              {!hash && (
                <>
                  <p>{t("receive_email")}</p>
                  <b>{t("steps")}</b>
                  <ol>
                    <li>{t("step_1")}</li>
                    <li>{t("step_2")}</li>
                    <li>{t("step_3")}</li>
                    <li>{t("step_4")}</li>
                    <li>{t("step_5")}</li>
                  </ol>
                </>
              )}
              {hash && (
                <>
                  <p>{t("step_4")}</p>
                </>
              )}
              <div
                style={{
                  display: "flex",
                  justifyContent: "center",
                  marginTop: "40px",
                }}
              >
                <div id="yivi-web-form"></div>
              </div>
            </div>
          </div>
        </main>
        <footer>
          <div className="actions">
            {!hash && (
              <Link to={`/${i18n.language}/validate`} id="back-button">
                {t("back")}
              </Link>
            )}
          </div>
        </footer>
      </div>
    </>
  );
}
