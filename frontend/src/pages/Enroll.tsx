import { useTranslation } from "react-i18next";
import { Link, useNavigate, useLocation } from "react-router-dom";
import { useAppContext } from "../AppContext";
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
  const hash = window.location.hash;
  const location = useLocation();
  const [token, setToken] = useState("");
  const { email, setEmail } = useAppContext();

  useEffect(() => {
    // only show the message if the user came from the validate page
    if (location.state?.from === "validate" && location.state?.message) {
      setMessage(t(location.state.message));

      window.history.replaceState({}, document.title);
    }
  }, [location.state]);
  // user clicks the link in the email to verify and start the issuance
  const VerifyAndStartIssuance = async (email: string, token: string) => {
    try {
      // send email and token to verify endpoint to see if the token is valid for this email
      const response = await fetch("/api/verify", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          email: email,
          token: token,
        }),
      });

      if (response.ok) {
        // Start enrollment process
        const res: VerifyResponse = await response.json();

        import("@privacybydesign/yivi-frontend")
          .then((yivi) => {
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
                navigate(`/${i18n.language}/done`);
              })
              .catch((e: string) => {
                if (e === "Aborted") {
                  setErrorMessage(t("email_add_cancel"));
                } else {
                  setErrorMessage(t("email_add_error"));
                }
              });
          })
          .finally(() => {
            fetch("/api/done", {
              method: "POST",
              headers: {
                "Content-Type": "application/json",
              },
              body: JSON.stringify({ email: email }),
            });
          });

        return;
      } else {
        const data = await response.json();
        let errorCode = data.error;

        if (errorCode) {
          setErrorMessage(t(errorCode));
        } else {
          navigate(`/${i18n.language}/error`);
        }
      }
    } catch (error) {
      console.error(error);
      navigate(`/${i18n.language}/error`);
    }
  };

  const enroll = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setErrorMessage(undefined);

    if (!token || token.length !== 6 || !email) {
      navigate(`/${i18n.language}/error`);
      return;
    }

    await VerifyAndStartIssuance(email, token);
  };

  useEffect(() => {
    if (hash) {
      const match = hash.match(/^#verify:([^:]+@[^\s:]+):(.+)$/);
      if (!match) {
        navigate(`/${i18n.language}/error`);
        return;
      }

      const email = match[1];
      const token = match[2];
      VerifyAndStartIssuance(email, token);
    }
  }, [navigate, t]);

  return (
    <>
      <form id="container" onSubmit={enroll}>
        <header>
          <h1>{t("index_header")}</h1>
        </header>
        <main>
          <div className="email-form">
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
                  </ol>
                  <label htmlFor="token">{t("enter_verification_code")}</label>
                  <input
                    type="text"
                    required
                    className="form-control verification-code-input"
                    value={token}
                    pattern="[0-9A-Za-z]{6}"
                    style={{ textTransform: "uppercase" }}
                    onChange={(e) => setToken(e.target.value.toUpperCase())}
                    autoFocus
                  />
                  <button
                    className="hidden"
                    id="submit-token"
                    type="submit"
                  ></button>
                </>
              )}
              {hash && (
                <>
                  <p>{t("step_4")}</p>
                </>
              )}
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
      </form>
    </>
  );
}
