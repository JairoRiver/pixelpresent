import { useEffect, useState } from 'preact/hooks';
import { isLoggedIn } from '../lib/session';

interface Props {
  loggedInHref: string;
  loggedInLabel: string;
  loggedOutHref: string;
  loggedOutLabel: string;
  class?: string;
}

// AuthLink renders a different target and label depending on whether there is a
// session. Until the check resolves it shows the logged-out variant — the safe
// default, and what a no-JS visitor gets.
export default function AuthLink(props: Props) {
  const [authed, setAuthed] = useState(false);

  useEffect(() => {
    isLoggedIn().then(setAuthed);
  }, []);

  return (
    <a href={authed ? props.loggedInHref : props.loggedOutHref} class={props.class}>
      {authed ? props.loggedInLabel : props.loggedOutLabel}
    </a>
  );
}
