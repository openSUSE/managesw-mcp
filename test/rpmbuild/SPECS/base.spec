Name:           base
Version:        1.0
Release:        1
Summary:        A base package
License:        MIT
Prefix:         /usr
AutoReqProv:    no

%description
This is a base package.

%install
mkdir -p %{buildroot}/%{_prefix}/bin
cat << EOF > %{buildroot}/%{_prefix}/bin/base_command
#!/bin/sh
echo "Hello from base"
EOF
chmod +x %{buildroot}/%{_prefix}/bin/base_command

%files
%{_prefix}/bin/base_command

%changelog
* Thu Jul 02 2026 Chris <chris@opensuse.org> - 1.0-1
- Line 1 of changelog
- Line 2 of changelog
- Line 3 of changelog
- Line 4 of changelog