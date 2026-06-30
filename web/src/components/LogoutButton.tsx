interface Props {
  class?: string;
}

// LogoutButton clears the session cookie and returns to the landing.
export default function LogoutButton(props: Props) {
  async function logout() {
    try {
      await fetch('/api/auth/logout', { method: 'POST' });
    } finally {
      window.location.href = '/';
    }
  }

  return (
    <button type="button" onClick={() => void logout()} class={props.class}>
      Cerrar sesión
    </button>
  );
}
