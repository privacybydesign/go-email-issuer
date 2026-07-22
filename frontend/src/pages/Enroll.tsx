import { useTranslation } from "react-i18next";
import { Link, useNavigate, useLocation } from "react-router-dom";
import { useAppContext } from "../AppContext";
import i18n from "../i18n";
import { useEffect, useRef, useState } from "react";
type VerifyResponse = {
  jwt: string;
  irma_server_url: string;
  // Only present on the verification-link flow: the email is resolved
  // server-side from the opaque token and returned here so issuance can finish.
  email?: string;
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
  const [token, setToken] = useState("");
  const { email, setEmail } = useAppContext();
  // Guard against the link-verify effect firing more than once. The link token
  // is single-use server-side, so a second run (e.g. React.StrictMode's
  // double-invoke in dev) would fail with error_token_invalid and show a
  // spurious error banner next to the launched Yivi popup.
  const linkVerifyStarted = useRef(false);

  useEffect(() => {
    // only show the message if the user came from the validate page
    if (location.state?.from === "validate" && location.state?.message) {
      setMessage(t(location.state.message));

      window.history.replaceState({}, document.title);
    }
  }, [location.state]);
  // Launch the Yivi issuance popup for a verified email and clean up afterwards.
  const startIssuance = (res: VerifyResponse, emailForCleanup: string) => {
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
          body: JSON.stringify({ email: emailForCleanup }),
        });
      });
  };

  const handleVerifyError = async (response: Response) => {
    try {
      const data = await response.json();
      if (data.error) {
        setErrorMessage(t(data.error));
        return;
      }
    } catch {
      // fall through to the generic error page
    }
    navigate(`/${i18n.language}/error`);
  };

  // Manual flow: the user types the 6-character code shown in the email.
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
        const res: VerifyResponse = await response.json();
        startIssuance(res, email);
        return;
      }
      await handleVerifyError(response);
    } catch (error) {
      console.error(error);
      navigate(`/${i18n.language}/error`);
    }
  };

  // Link flow: the user clicks the verification link, which carries only an
  // opaque token. The email is resolved server-side, so it never appears in the
  // URL or browser history (issue #44).
  const verifyLinkAndStartIssuance = async (linkToken: string) => {
    try {
      const response = await fetch("/api/verify-link", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          link_token: linkToken,
        }),
      });

      if (response.ok) {
        const res: VerifyResponse = await response.json();
        if (res.email) {
          setEmail(res.email);
        }
        startIssuance(res, res.email ?? "");
        return;
      }
      await handleVerifyError(response);
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
      if (linkVerifyStarted.current) {
        return;
      }
      linkVerifyStarted.current = true;

      const match = hash.match(/^#token:(.+)$/);
      if (!match) {
        navigate(`/${i18n.language}/error`);
        return;
      }

      const linkToken = match[1];

      // Strip the token from the URL immediately so this bearer credential is
      // not persisted in browser history on shared/public devices (issue #44).
      window.history.replaceState(
        null,
        document.title,
        window.location.pathname + window.location.search
      );

      verifyLinkAndStartIssuance(linkToken);
    }
  }, [navigate, t, hash]);

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
            <button type="submit" id="next-button">
              {t("verify")}
            </button>
          </div>
        </footer>
      </form>
    </>
  );
}
