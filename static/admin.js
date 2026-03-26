(function (window, document, shared) {
    const ROLE_USER = "user";
    const ROLE_ADMIN = "admin";
    const ADMIN_DENIED_MESSAGE = "доступ только для администратора";

    let currentUser = null;

    const elements = {
        adminIdentity: document.getElementById("adminIdentity"),
        adminNotice: document.getElementById("adminNotice"),
        adminState: document.getElementById("adminState"),
        authState: document.getElementById("authState"),
        forbiddenMessage: document.getElementById("forbiddenMessage"),
        forbiddenState: document.getElementById("forbiddenState"),
        homeLink: document.getElementById("homeLink"),
        likesList: document.getElementById("likesList"),
        loadingState: document.getElementById("loadingState"),
        logoutButton: document.getElementById("logoutButton"),
        refreshButton: document.getElementById("refreshButton"),
        usersTableBody: document.getElementById("usersTableBody")
    };

    function showState(stateId) {
        ["loadingState", "authState", "forbiddenState", "adminState"].forEach((id) => {
            document.getElementById(id).classList.toggle("hidden", id !== stateId);
        });
    }

    function setNotice(message, isError) {
        if (!message) {
            elements.adminNotice.textContent = "";
            elements.adminNotice.classList.add("hidden");
            elements.adminNotice.classList.remove("error");
            return;
        }

        elements.adminNotice.textContent = message;
        elements.adminNotice.classList.remove("hidden");
        elements.adminNotice.classList.toggle("error", Boolean(isError));
    }

    function showAuthState() {
        currentUser = null;
        showState("authState");
        elements.adminIdentity.classList.add("hidden");
        elements.refreshButton.classList.add("hidden");
        elements.logoutButton.classList.add("hidden");
    }

    function showForbiddenState(message) {
        showState("forbiddenState");
        elements.forbiddenMessage.textContent = message || ADMIN_DENIED_MESSAGE;
        elements.adminIdentity.classList.add("hidden");
        elements.refreshButton.classList.add("hidden");
        elements.logoutButton.classList.add("hidden");
    }

    function showAdminState() {
        showState("adminState");
        elements.adminIdentity.textContent = `admin: ${currentUser.username}`;
        elements.adminIdentity.classList.remove("hidden");
        elements.refreshButton.classList.remove("hidden");
        elements.logoutButton.classList.remove("hidden");
    }

    function api(endpoint, options) {
        return shared.api(endpoint, options, {
            onResponse(response, payload) {
                if (response.status === 401) {
                    showAuthState();
                    return;
                }
                if (response.status === 403 && payload && payload.message === ADMIN_DENIED_MESSAGE) {
                    showForbiddenState(payload.message);
                }
            }
        });
    }

    async function logout() {
        await api("/logout", { method: "POST" });
        window.location.href = shared.homeUrl();
    }

    function roleLabel(user) {
        if (!user.is_allowed) {
            return '<span class="role-pill guest">не разрешён</span>';
        }
        if (user.role === ROLE_ADMIN) {
            return '<span class="role-pill admin">admin</span>';
        }
        return '<span class="role-pill user">user</span>';
    }

    function renderUserActions(user) {
        if (!user.is_allowed) {
            return `<button class="btn btn-primary" type="button" data-action="allow-user" data-user-id="${user.id}">Разрешить</button>`;
        }
        if (user.role === ROLE_ADMIN) {
            return `<button class="btn btn-secondary" type="button" data-action="change-role" data-user-id="${user.id}" data-role="${ROLE_USER}">Сделать user</button>`;
        }
        return [
            `<button class="btn btn-primary" type="button" data-action="change-role" data-user-id="${user.id}" data-role="${ROLE_ADMIN}">Сделать admin</button>`,
            `<button class="btn btn-danger" type="button" data-action="remove-allowed" data-user-id="${user.id}">Удалить доступ</button>`
        ].join("");
    }

    function renderUsers(users) {
        if (!users.length) {
            elements.usersTableBody.innerHTML = '<tr><td colspan="4"><div class="empty">Пользователей пока нет.</div></td></tr>';
            return;
        }

        elements.usersTableBody.innerHTML = users.map((user) => {
            const meta = `<div class="user-meta">id ${user.id} · создан ${shared.formatDate(user.created_at, { includeYear: true })}</div>`;
            return `<tr>
                <td><strong>${shared.esc(user.username)}</strong>${meta}</td>
                <td>${user.post_count}</td>
                <td>${roleLabel(user)}</td>
                <td><div class="actions">${renderUserActions(user)}</div></td>
            </tr>`;
        }).join("");
    }

    function renderLikes(posts) {
        if (!posts.length) {
            elements.likesList.innerHTML = '<div class="empty">Пока никто ничего не лайкал.</div>';
            return;
        }

        elements.likesList.innerHTML = posts.map((post) => {
            const content = post.content
                ? shared.esc(post.content)
                : '<span class="post-content-empty">Пост без текста</span>';

            const flags = [
                `<span class="flag">Лайков: ${post.like_count}</span>`,
                post.image_url ? '<span class="flag">Есть картинка</span>' : ""
            ].join("");

            const likedUsers = post.liked_users.map((user) => {
                return `<div class="liked-user">
                    <strong>${shared.esc(user.username)}</strong>
                    <time>${shared.formatDate(user.liked_at, { includeYear: true })}</time>
                </div>`;
            }).join("");

            return `<article class="post-card">
                <h3>Пост #${post.post_id} · ${shared.esc(post.author_username)}</h3>
                <div class="post-meta">${shared.formatDate(post.created_at, { includeYear: true })}</div>
                <div class="post-content">${content}</div>
                <div class="post-flags">${flags}</div>
                <div class="liked-users">${likedUsers}</div>
            </article>`;
        }).join("");
    }

    async function loadAdminData() {
        if (!currentUser || currentUser.role !== ROLE_ADMIN) {
            return;
        }

        setNotice("");
        const [usersResponse, likesResponse] = await Promise.all([
            api("/admin/users"),
            api("/admin/likes")
        ]);

        if (!usersResponse.success || !likesResponse.success) {
            if (usersResponse.status === 401 || likesResponse.status === 401) {
                showAuthState();
                return;
            }
            if (usersResponse.status === 403 || likesResponse.status === 403) {
                showForbiddenState(ADMIN_DENIED_MESSAGE);
                return;
            }
            setNotice(usersResponse.message || likesResponse.message || "Не удалось загрузить данные админки", true);
            return;
        }

        renderUsers(usersResponse.data || []);
        renderLikes(likesResponse.data || []);
    }

    async function mutateAdminData(endpoint, payload, successMessage) {
        setNotice("");
        const response = await api(endpoint, {
            method: "POST",
            body: payload ? JSON.stringify(payload) : undefined
        });

        if (!response.success) {
            if (response.status !== 401 && response.status !== 403) {
                setNotice(response.message || "Операция не выполнена", true);
            }
            return;
        }

        await loadAdminData();
        setNotice(successMessage);
    }

    async function bootstrap() {
        const meResponse = await api("/me");
        if (!meResponse.success || !meResponse.data) {
            if (meResponse.status === 401) {
                showAuthState();
                return;
            }
            showForbiddenState("Не удалось загрузить пользователя");
            return;
        }

        currentUser = meResponse.data;
        if (currentUser.role !== ROLE_ADMIN) {
            showForbiddenState(ADMIN_DENIED_MESSAGE);
            return;
        }

        showAdminState();
        await loadAdminData();
    }

    function bindEvents() {
        elements.refreshButton.addEventListener("click", loadAdminData);
        elements.logoutButton.addEventListener("click", logout);

        shared.delegate(elements.usersTableBody, "click", "[data-action='allow-user']", (event, button) => {
            event.preventDefault();
            mutateAdminData("/admin/allowed-users", { user_id: Number(button.dataset.userId) }, "Пользователь добавлен в разрешенные");
        });

        shared.delegate(elements.usersTableBody, "click", "[data-action='change-role']", (event, button) => {
            event.preventDefault();
            const role = button.dataset.role;
            const label = role === ROLE_ADMIN ? "Пользователь повышен до admin" : "Администратор понижен до user";
            mutateAdminData(`/admin/allowed-users/${Number(button.dataset.userId)}/role`, { role }, label);
        });

        shared.delegate(elements.usersTableBody, "click", "[data-action='remove-allowed']", (event, button) => {
            event.preventDefault();
            mutateAdminData(`/admin/allowed-users/${Number(button.dataset.userId)}/remove`, null, "Доступ пользователя удалён");
        });
    }

    function init() {
        elements.homeLink.href = shared.homeUrl();
        bindEvents();
        bootstrap();
    }

    init();
})(window, document, window.KuzovokShared);
